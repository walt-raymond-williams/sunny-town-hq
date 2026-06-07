package inventory

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUnknownRecipe          = errors.New("unknown recipe")
	ErrInsufficientIngredient = errors.New("not enough ingredients")
)

type CraftRecipeRequest struct {
	RecipeKey string `json:"recipeKey"`
}

type CraftingIngredientResponse struct {
	ItemKey     string `json:"itemKey"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    int    `json:"required"`
	Owned       int    `json:"owned"`
}

type CraftingRecipeResponse struct {
	Key         string                       `json:"key"`
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	OutputKey   string                       `json:"outputKey"`
	OutputName  string                       `json:"outputName"`
	Quantity    int                          `json:"quantity"`
	CanCraft    bool                         `json:"canCraft"`
	Ingredients []CraftingIngredientResponse `json:"ingredients"`
}

type CraftingRecipesResponse struct {
	Recipes []CraftingRecipeResponse `json:"recipes"`
}

type CraftRecipeResponse struct {
	Inventory StudentResponse          `json:"inventory"`
	Recipes   []CraftingRecipeResponse `json:"recipes"`
}

type craftingRecipe struct {
	Key         string
	OutputKey   string
	Quantity    int
	Ingredients []craftingIngredient
}

type craftingIngredient struct {
	ItemKey  string
	Quantity int
}

var craftingRecipes = []craftingRecipe{
	{
		Key:       "stone_block",
		OutputKey: "stone_block",
		Quantity:  1,
		Ingredients: []craftingIngredient{
			{ItemKey: "rock", Quantity: 4},
		},
	},
}

func LoadCraftingRecipes(ctx context.Context, querier Loader, userID int64) (CraftingRecipesResponse, error) {
	itemKeys := map[string]bool{}
	for _, recipe := range craftingRecipes {
		itemKeys[recipe.OutputKey] = true
		for _, ingredient := range recipe.Ingredients {
			itemKeys[ingredient.ItemKey] = true
		}
	}

	ownedItems, err := loadCraftingItemMetadata(ctx, querier, userID, itemKeys)
	if err != nil {
		return CraftingRecipesResponse{}, err
	}

	response := CraftingRecipesResponse{Recipes: []CraftingRecipeResponse{}}
	for _, recipe := range craftingRecipes {
		output := ownedItems[recipe.OutputKey]
		recipeResponse := CraftingRecipeResponse{
			Key:         recipe.Key,
			Name:        output.Name,
			Description: output.Description,
			OutputKey:   recipe.OutputKey,
			OutputName:  output.Name,
			Quantity:    recipe.Quantity,
			CanCraft:    true,
			Ingredients: []CraftingIngredientResponse{},
		}
		for _, ingredient := range recipe.Ingredients {
			item := ownedItems[ingredient.ItemKey]
			if item.Quantity < ingredient.Quantity {
				recipeResponse.CanCraft = false
			}
			recipeResponse.Ingredients = append(recipeResponse.Ingredients, CraftingIngredientResponse{
				ItemKey:     ingredient.ItemKey,
				Name:        item.Name,
				Description: item.Description,
				Required:    ingredient.Quantity,
				Owned:       item.Quantity,
			})
		}
		response.Recipes = append(response.Recipes, recipeResponse)
	}
	return response, nil
}

func CraftStudentRecipe(ctx context.Context, db *pgxpool.Pool, userID int64, request CraftRecipeRequest) (CraftRecipeResponse, error) {
	request.RecipeKey = strings.TrimSpace(request.RecipeKey)
	if userID < 1 || request.RecipeKey == "" {
		return CraftRecipeResponse{}, errors.New("crafting request is missing required fields")
	}

	recipe, ok := craftingRecipeByKey(request.RecipeKey)
	if !ok {
		return CraftRecipeResponse{}, ErrUnknownRecipe
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return CraftRecipeResponse{}, err
	}
	defer tx.Rollback(ctx)

	for _, ingredient := range recipe.Ingredients {
		consumed, err := ConsumeStudentItem(ctx, tx, userID, ingredient.ItemKey, ingredient.Quantity)
		if err != nil {
			return CraftRecipeResponse{}, err
		}
		if !consumed {
			return CraftRecipeResponse{}, ErrInsufficientIngredient
		}
	}

	if err := IncrementStudentItem(ctx, tx, userID, recipe.OutputKey, recipe.Quantity); err != nil {
		return CraftRecipeResponse{}, err
	}

	inventory, err := LoadStudent(ctx, tx, userID)
	if err != nil {
		return CraftRecipeResponse{}, err
	}
	recipes, err := LoadCraftingRecipes(ctx, tx, userID)
	if err != nil {
		return CraftRecipeResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CraftRecipeResponse{}, err
	}

	return CraftRecipeResponse{
		Inventory: inventory,
		Recipes:   recipes.Recipes,
	}, nil
}

func CraftingErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrUnknownRecipe):
		return "recipe not found"
	case errors.Is(err, ErrInsufficientIngredient):
		return "not enough ingredients"
	default:
		return "crafting could not be completed"
	}
}

func craftingRecipeByKey(key string) (craftingRecipe, bool) {
	for _, recipe := range craftingRecipes {
		if recipe.Key == key {
			return recipe, true
		}
	}
	return craftingRecipe{}, false
}

type craftingItemMetadata struct {
	Name        string
	Description string
	Quantity    int
}

func loadCraftingItemMetadata(ctx context.Context, querier Loader, userID int64, itemKeys map[string]bool) (map[string]craftingItemMetadata, error) {
	keys := make([]string, 0, len(itemKeys))
	for key := range itemKeys {
		keys = append(keys, key)
	}

	rows, err := querier.Query(
		ctx,
		`
			select iit.key,
				iit.name,
				iit.description,
				coalesce(sii.quantity, 0) as quantity
			from inventory_item_type iit
			left join student_inventory_item sii on sii.item_type_id = iit.id
				and sii.app_user_id = $1
			where iit.key = any($2)
		`,
		userID,
		keys,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := map[string]craftingItemMetadata{}
	for rows.Next() {
		var key string
		var item craftingItemMetadata
		if err := rows.Scan(&key, &item.Name, &item.Description, &item.Quantity); err != nil {
			return nil, err
		}
		items[key] = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
