import { describe, expect, it } from 'vitest'
import type { CraftingRecipe, InventoryItem } from '../types/inventory'
import { withKnownCraftingRecipes } from './craftingRecipes'

function item(overrides: Partial<InventoryItem>): InventoryItem {
  return {
    key: 'rock',
    name: 'Rock',
    description: 'A sturdy rock.',
    quantity: 0,
    equipSlot: '',
    visualKey: '',
    iconKey: 'rock',
    maxStack: 64,
    category: 'resource',
    equipped: false,
    ...overrides,
  }
}

describe('withKnownCraftingRecipes', () => {
  it('adds a craftable stone block recipe from inventory rocks when the API omits it', () => {
    const recipes = withKnownCraftingRecipes([], [item({ key: 'rock', quantity: 4 })])

    expect(recipes).toHaveLength(1)
    expect(recipes[0]).toMatchObject({
      key: 'stone_block',
      outputKey: 'stone_block',
      canCraft: true,
    })
    expect(recipes[0]?.ingredients[0]).toMatchObject({
      itemKey: 'rock',
      owned: 4,
      required: 4,
    })
  })

  it('keeps the server recipe when it is present', () => {
    const serverRecipe: CraftingRecipe = {
      key: 'stone_block',
      name: 'Server Stone Block',
      description: '',
      outputKey: 'stone_block',
      outputName: 'Server Stone Block',
      outputIconKey: 'stone_block',
      outputMaxStack: 64,
      outputCategory: 'building',
      quantity: 1,
      canCraft: false,
      ingredients: [],
    }

    expect(withKnownCraftingRecipes([serverRecipe], [item({ quantity: 4 })])).toEqual([serverRecipe])
  })
})
