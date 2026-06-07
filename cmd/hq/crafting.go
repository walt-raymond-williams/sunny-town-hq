package main

import (
	"context"

	hqinventory "hq/internal/hq/inventory"
)

var (
	errUnknownRecipe          = hqinventory.ErrUnknownRecipe
	errInsufficientIngredient = hqinventory.ErrInsufficientIngredient
)

type craftRecipeRequest = hqinventory.CraftRecipeRequest
type craftingIngredientResponse = hqinventory.CraftingIngredientResponse
type craftingRecipeResponse = hqinventory.CraftingRecipeResponse
type craftingRecipesResponse = hqinventory.CraftingRecipesResponse
type craftRecipeResponse = hqinventory.CraftRecipeResponse

func (app *app) loadCraftingRecipes(ctx context.Context, userID int64) (craftingRecipesResponse, error) {
	return loadCraftingRecipes(ctx, app.db, userID)
}

func loadCraftingRecipes(ctx context.Context, querier inventoryLoader, userID int64) (craftingRecipesResponse, error) {
	return hqinventory.LoadCraftingRecipes(ctx, querier, userID)
}

func (app *app) craftStudentRecipe(ctx context.Context, userID int64, request craftRecipeRequest) (craftRecipeResponse, error) {
	return hqinventory.CraftStudentRecipe(ctx, app.db, userID, request)
}

func craftingErrorMessage(err error) string {
	return hqinventory.CraftingErrorMessage(err)
}
