import type { CraftingRecipe, InventoryItem } from '../types/inventory'

export function withKnownCraftingRecipes(recipes: CraftingRecipe[], items: InventoryItem[]): CraftingRecipe[] {
  if (recipes.some((recipe) => recipe.key === 'stone_block')) {
    return recipes
  }

  const rockQuantity = items.find((item) => item.key === 'rock')?.quantity || 0
  return [
    ...recipes,
    {
      key: 'stone_block',
      name: 'Stone Block',
      description: 'A solid block crafted from stone.',
      outputKey: 'stone_block',
      outputName: 'Stone Block',
      outputIconKey: 'stone_block',
      outputMaxStack: 64,
      outputCategory: 'building',
      quantity: 1,
      canCraft: rockQuantity >= 4,
      ingredients: [
        {
          itemKey: 'rock',
          name: 'Rock',
          description: 'A sturdy rock from Forest Crossing.',
          iconKey: 'rock',
          maxStack: 64,
          category: 'resource',
          required: 4,
          owned: rockQuantity,
        },
      ],
    },
  ]
}
