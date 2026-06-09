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
	ErrRecipeOutputFull       = errors.New("recipe output storage is full")
)

const (
	CookieRecipeKey = "cookie"
	FlourKey        = "flour"
	SugarKey        = "sugar"
)

type CraftRecipeRequest struct {
	RecipeKey string `json:"recipeKey"`
}

type CraftingIngredientResponse struct {
	ItemKey     string `json:"itemKey"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IconKey     string `json:"iconKey,omitempty"`
	MaxStack    int    `json:"maxStack,omitempty"`
	Category    string `json:"category,omitempty"`
	Required    int    `json:"required"`
	Owned       int    `json:"owned"`
}

type CraftingRecipeResponse struct {
	Key            string                       `json:"key"`
	Name           string                       `json:"name"`
	Description    string                       `json:"description"`
	OutputKey      string                       `json:"outputKey"`
	OutputName     string                       `json:"outputName"`
	OutputIconKey  string                       `json:"outputIconKey,omitempty"`
	OutputMaxStack int                          `json:"outputMaxStack,omitempty"`
	OutputCategory string                       `json:"outputCategory,omitempty"`
	Quantity       int                          `json:"quantity"`
	CanCraft       bool                         `json:"canCraft"`
	Ingredients    []CraftingIngredientResponse `json:"ingredients"`
}

type CraftingRecipesResponse struct {
	Recipes []CraftingRecipeResponse `json:"recipes"`
}

type CraftRecipeResponse struct {
	Inventory StudentInventorySlotsResponse `json:"inventory"`
	Recipes   []CraftingRecipeResponse      `json:"recipes"`
}

type RecipeDefinition struct {
	Key                  string
	OutputKey            string
	Quantity             int
	Ingredients          []RecipeIngredient
	AvailableToStudentUI bool
}

type RecipeIngredient struct {
	ItemKey  string
	Quantity int
}

type recipeStorage interface {
	ConsumeRecipeItem(ctx context.Context, itemKey string, quantity int) (bool, error)
	ProduceRecipeItem(ctx context.Context, itemKey string, quantity int) error
}

type recipePreparingStorage interface {
	PrepareRecipe(ctx context.Context, recipe RecipeDefinition) error
}

type studentRecipeStorage struct {
	querier Querier
	userID  int64
}

var recipeCatalog = []RecipeDefinition{
	{
		Key:                  "stone_block",
		OutputKey:            "stone_block",
		Quantity:             1,
		AvailableToStudentUI: true,
		Ingredients: []RecipeIngredient{
			{ItemKey: "rock", Quantity: 4},
		},
	},
	{
		Key:       CookieRecipeKey,
		OutputKey: CookieKey,
		Quantity:  1,
		Ingredients: []RecipeIngredient{
			{ItemKey: FlourKey, Quantity: 1},
			{ItemKey: SugarKey, Quantity: 1},
		},
	},
}

func LoadCraftingRecipes(ctx context.Context, querier Loader, userID int64) (CraftingRecipesResponse, error) {
	itemKeys := map[string]bool{}
	for _, recipe := range recipeCatalog {
		if !recipe.AvailableToStudentUI {
			continue
		}
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
	for _, recipe := range recipeCatalog {
		if !recipe.AvailableToStudentUI {
			continue
		}
		output := ownedItems[recipe.OutputKey]
		recipeResponse := CraftingRecipeResponse{
			Key:            recipe.Key,
			Name:           output.Name,
			Description:    output.Description,
			OutputKey:      recipe.OutputKey,
			OutputName:     output.Name,
			OutputIconKey:  output.IconKey,
			OutputMaxStack: output.MaxStack,
			OutputCategory: output.Category,
			Quantity:       recipe.Quantity,
			CanCraft:       true,
			Ingredients:    []CraftingIngredientResponse{},
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
				IconKey:     item.IconKey,
				MaxStack:    item.MaxStack,
				Category:    item.Category,
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

	recipe, ok := recipeByKey(request.RecipeKey)
	if !ok {
		return CraftRecipeResponse{}, ErrUnknownRecipe
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return CraftRecipeResponse{}, err
	}
	defer tx.Rollback(ctx)

	storage := studentRecipeStorage{querier: tx, userID: userID}
	if err := executeRecipe(ctx, recipe, storage); err != nil {
		return CraftRecipeResponse{}, err
	}

	inventory, err := LoadStudentSlots(ctx, tx, userID)
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
	case errors.Is(err, ErrInventoryFull):
		return "not enough room in inventory"
	default:
		return "crafting could not be completed"
	}
}

func recipeByKey(key string) (RecipeDefinition, bool) {
	for _, recipe := range recipeCatalog {
		if recipe.Key == key {
			return recipe, true
		}
	}
	return RecipeDefinition{}, false
}

func executeRecipe(ctx context.Context, recipe RecipeDefinition, storage recipeStorage) error {
	if preparingStorage, ok := storage.(recipePreparingStorage); ok {
		if err := preparingStorage.PrepareRecipe(ctx, recipe); err != nil {
			return err
		}
	}

	for _, ingredient := range recipe.Ingredients {
		consumed, err := storage.ConsumeRecipeItem(ctx, ingredient.ItemKey, ingredient.Quantity)
		if err != nil {
			return err
		}
		if !consumed {
			return ErrInsufficientIngredient
		}
	}

	return storage.ProduceRecipeItem(ctx, recipe.OutputKey, recipe.Quantity)
}

func scaledRecipe(recipe RecipeDefinition, amount int) RecipeDefinition {
	if amount <= 1 {
		return recipe
	}
	scaled := recipe
	scaled.Quantity = recipe.Quantity * amount
	scaled.Ingredients = make([]RecipeIngredient, 0, len(recipe.Ingredients))
	for _, ingredient := range recipe.Ingredients {
		scaled.Ingredients = append(scaled.Ingredients, RecipeIngredient{
			ItemKey:  ingredient.ItemKey,
			Quantity: ingredient.Quantity * amount,
		})
	}
	return scaled
}

func (storage studentRecipeStorage) ConsumeRecipeItem(ctx context.Context, itemKey string, quantity int) (bool, error) {
	return ConsumeStudentItem(ctx, storage.querier, storage.userID, itemKey, quantity)
}

func (storage studentRecipeStorage) ProduceRecipeItem(ctx context.Context, itemKey string, quantity int) error {
	return IncrementStudentItem(ctx, storage.querier, storage.userID, itemKey, quantity)
}

type craftingItemMetadata struct {
	Name        string
	Description string
	IconKey     string
	MaxStack    int
	Category    string
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
				coalesce(iit.icon_key, '') as icon_key,
				coalesce(iit.max_stack, 0) as max_stack,
				coalesce(iit.category, '') as category,
				coalesce(slot_totals.quantity, 0) as quantity
			from inventory_item_type iit
			left join (
				select item_type_id, coalesce(sum(quantity), 0)::integer as quantity
				from student_inventory_slot
				where app_user_id = $1
				group by item_type_id
			) slot_totals on slot_totals.item_type_id = iit.id
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
		if err := rows.Scan(&key, &item.Name, &item.Description, &item.IconKey, &item.MaxStack, &item.Category, &item.Quantity); err != nil {
			return nil, err
		}
		items[key] = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
