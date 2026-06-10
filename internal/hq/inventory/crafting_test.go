package inventory

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type fakeRecipeStorage struct {
	consumeResults map[string]bool
	quantities     map[string]int
	consumed       []RecipeIngredient
	produced       []RecipeIngredient
}

func (storage *fakeRecipeStorage) RecipeInputDescriptor() RecipeStorageDescriptor {
	return RecipeStorageDescriptor{Kind: RecipeStorageKindMemory, Label: "test input storage"}
}

func (storage *fakeRecipeStorage) RecipeOutputDescriptor() RecipeStorageDescriptor {
	return RecipeStorageDescriptor{Kind: RecipeStorageKindMemory, Label: "test output storage"}
}

func (storage *fakeRecipeStorage) RecipeItemQuantity(ctx context.Context, itemKey string) (int, error) {
	return storage.quantities[itemKey], nil
}

func (storage *fakeRecipeStorage) ConsumeRecipeItem(ctx context.Context, itemKey string, quantity int) (bool, error) {
	storage.consumed = append(storage.consumed, RecipeIngredient{ItemKey: itemKey, Quantity: quantity})
	return storage.consumeResults[itemKey], nil
}

func (storage *fakeRecipeStorage) ProduceRecipeItem(ctx context.Context, itemKey string, quantity int) error {
	storage.produced = append(storage.produced, RecipeIngredient{ItemKey: itemKey, Quantity: quantity})
	return nil
}

func TestExecuteRecipeConsumesIngredientsAndProducesOutput(t *testing.T) {
	recipe := RecipeDefinition{
		Key:       "cookie",
		OutputKey: "cookie",
		Quantity:  2,
		Ingredients: []RecipeIngredient{
			{ItemKey: "flour", Quantity: 1},
			{ItemKey: "sugar", Quantity: 3},
		},
	}
	storage := &fakeRecipeStorage{consumeResults: map[string]bool{
		"flour": true,
		"sugar": true,
	}}

	if err := executeRecipe(context.Background(), recipe, storage); err != nil {
		t.Fatalf("executeRecipe error = %v", err)
	}

	expectedConsumed := []RecipeIngredient{
		{ItemKey: "flour", Quantity: 1},
		{ItemKey: "sugar", Quantity: 3},
	}
	if !reflect.DeepEqual(storage.consumed, expectedConsumed) {
		t.Fatalf("consumed = %#v, want %#v", storage.consumed, expectedConsumed)
	}
	expectedProduced := []RecipeIngredient{{ItemKey: "cookie", Quantity: 2}}
	if !reflect.DeepEqual(storage.produced, expectedProduced) {
		t.Fatalf("produced = %#v, want %#v", storage.produced, expectedProduced)
	}
}

func TestExecuteRecipeStopsBeforeOutputWhenIngredientMissing(t *testing.T) {
	recipe := RecipeDefinition{
		Key:       "cookie",
		OutputKey: "cookie",
		Quantity:  2,
		Ingredients: []RecipeIngredient{
			{ItemKey: "flour", Quantity: 1},
			{ItemKey: "sugar", Quantity: 3},
		},
	}
	storage := &fakeRecipeStorage{consumeResults: map[string]bool{
		"flour": true,
		"sugar": false,
	}}

	err := executeRecipe(context.Background(), recipe, storage)
	if !errors.Is(err, ErrInsufficientIngredient) {
		t.Fatalf("executeRecipe error = %v, want ErrInsufficientIngredient", err)
	}
	var storageErr recipeStorageError
	if !errors.As(err, &storageErr) {
		t.Fatalf("executeRecipe error = %T, want recipeStorageError", err)
	}
	if storageErr.Storage.Label != "test input storage" || storageErr.ItemKey != "sugar" {
		t.Fatalf("storage error = %#v, want sugar in test input storage", storageErr)
	}
	if len(storage.produced) != 0 {
		t.Fatalf("produced = %#v, want no output when ingredients are missing", storage.produced)
	}
}
