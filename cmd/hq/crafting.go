package main

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

var (
	errUnknownRecipe          = errors.New("unknown recipe")
	errInsufficientIngredient = errors.New("not enough ingredients")
)

type craftRecipeRequest struct {
	RecipeKey string `json:"recipeKey"`
}

type craftingIngredientResponse struct {
	ItemKey     string `json:"itemKey"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    int    `json:"required"`
	Owned       int    `json:"owned"`
}

type craftingRecipeResponse struct {
	Key         string                       `json:"key"`
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	OutputKey   string                       `json:"outputKey"`
	OutputName  string                       `json:"outputName"`
	Quantity    int                          `json:"quantity"`
	CanCraft    bool                         `json:"canCraft"`
	Ingredients []craftingIngredientResponse `json:"ingredients"`
}

type craftingRecipesResponse struct {
	Recipes []craftingRecipeResponse `json:"recipes"`
}

type craftRecipeResponse struct {
	Inventory studentInventoryResponse `json:"inventory"`
	Recipes   []craftingRecipeResponse `json:"recipes"`
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

func (app *app) loadCraftingRecipes(ctx context.Context, userID int64) (craftingRecipesResponse, error) {
	return loadCraftingRecipes(ctx, app.db, userID)
}

type craftingRecipeLoader interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func loadCraftingRecipes(ctx context.Context, querier craftingRecipeLoader, userID int64) (craftingRecipesResponse, error) {
	itemKeys := map[string]bool{}
	for _, recipe := range craftingRecipes {
		itemKeys[recipe.OutputKey] = true
		for _, ingredient := range recipe.Ingredients {
			itemKeys[ingredient.ItemKey] = true
		}
	}

	ownedItems, err := loadCraftingItemMetadata(ctx, querier, userID, itemKeys)
	if err != nil {
		return craftingRecipesResponse{}, err
	}

	response := craftingRecipesResponse{Recipes: []craftingRecipeResponse{}}
	for _, recipe := range craftingRecipes {
		output := ownedItems[recipe.OutputKey]
		recipeResponse := craftingRecipeResponse{
			Key:         recipe.Key,
			Name:        output.Name,
			Description: output.Description,
			OutputKey:   recipe.OutputKey,
			OutputName:  output.Name,
			Quantity:    recipe.Quantity,
			CanCraft:    true,
			Ingredients: []craftingIngredientResponse{},
		}
		for _, ingredient := range recipe.Ingredients {
			item := ownedItems[ingredient.ItemKey]
			if item.Quantity < ingredient.Quantity {
				recipeResponse.CanCraft = false
			}
			recipeResponse.Ingredients = append(recipeResponse.Ingredients, craftingIngredientResponse{
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

func (app *app) craftStudentRecipe(ctx context.Context, userID int64, request craftRecipeRequest) (craftRecipeResponse, error) {
	request.RecipeKey = strings.TrimSpace(request.RecipeKey)
	if userID < 1 || request.RecipeKey == "" {
		return craftRecipeResponse{}, errors.New("crafting request is missing required fields")
	}

	recipe, ok := craftingRecipeByKey(request.RecipeKey)
	if !ok {
		return craftRecipeResponse{}, errUnknownRecipe
	}

	tx, err := app.db.Begin(ctx)
	if err != nil {
		return craftRecipeResponse{}, err
	}
	defer tx.Rollback(ctx)

	for _, ingredient := range recipe.Ingredients {
		consumed, err := consumeStudentInventoryItem(ctx, tx, userID, ingredient.ItemKey, ingredient.Quantity)
		if err != nil {
			return craftRecipeResponse{}, err
		}
		if !consumed {
			return craftRecipeResponse{}, errInsufficientIngredient
		}
	}

	if err := incrementStudentInventoryItem(ctx, tx, userID, recipe.OutputKey, recipe.Quantity); err != nil {
		return craftRecipeResponse{}, err
	}

	inventory, err := loadStudentInventory(ctx, tx, userID)
	if err != nil {
		return craftRecipeResponse{}, err
	}
	recipes, err := loadCraftingRecipes(ctx, tx, userID)
	if err != nil {
		return craftRecipeResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return craftRecipeResponse{}, err
	}

	return craftRecipeResponse{
		Inventory: inventory,
		Recipes:   recipes.Recipes,
	}, nil
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

func loadCraftingItemMetadata(ctx context.Context, querier craftingRecipeLoader, userID int64, itemKeys map[string]bool) (map[string]craftingItemMetadata, error) {
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

func craftingErrorMessage(err error) string {
	switch {
	case errors.Is(err, errUnknownRecipe):
		return "recipe not found"
	case errors.Is(err, errInsufficientIngredient):
		return "not enough ingredients"
	default:
		return "crafting could not be completed"
	}
}
