import { describe, expect, it, vi } from 'vitest'

vi.mock('./http', () => ({
  authJson: vi.fn(),
  jsonOptions: vi.fn(),
}))

import { normalizeCraftingRecipe } from './craftingApi'
import { normalizeEquipmentItem } from './equipmentApi'
import { normalizeHotbar } from './hotbarApi'
import { normalizeInventoryItem } from './inventoryApi'

describe('inventory item metadata normalizers', () => {
  it('preserves inventory metadata and falls back iconKey to key', () => {
    expect(normalizeInventoryItem({
      key: 'stone_block',
      name: 'Stone Block',
      description: 'A solid block.',
      quantity: 2,
      iconKey: 'stone_block',
      maxStack: 64,
      category: 'building',
    })).toMatchObject({
      key: 'stone_block',
      iconKey: 'stone_block',
      maxStack: 64,
      category: 'building',
    })

    expect(normalizeInventoryItem({ key: 'rock' }).iconKey).toBe('rock')
  })

  it('preserves hotbar item metadata through shared inventory normalization', () => {
    const hotbar = normalizeHotbar({
      slots: [{
        slot: 1,
        item: {
          key: 'pickaxe',
          name: 'Pickaxe',
          quantity: 1,
          iconKey: 'pickaxe',
          maxStack: 1,
          category: 'tool',
        },
      }],
    })

    const firstSlot = hotbar.slots[0]
    expect(firstSlot).toBeDefined()
    expect(firstSlot?.item).toMatchObject({
      key: 'pickaxe',
      iconKey: 'pickaxe',
      maxStack: 1,
      category: 'tool',
    })
  })

  it('preserves equipment metadata separately from visualKey', () => {
    expect(normalizeEquipmentItem({
      key: 'sunny_hoodie',
      name: 'Sunny Hoodie',
      equipSlot: 'gear',
      visualKey: 'sunny_hoodie_visual',
      iconKey: 'sunny_hoodie',
      maxStack: 1,
      category: 'gear',
    })).toMatchObject({
      key: 'sunny_hoodie',
      visualKey: 'sunny_hoodie_visual',
      iconKey: 'sunny_hoodie',
      maxStack: 1,
      category: 'gear',
    })
  })

  it('preserves crafting output and ingredient metadata', () => {
    const recipe = normalizeCraftingRecipe({
      key: 'stone_block',
      outputKey: 'stone_block',
      outputName: 'Stone Block',
      outputIconKey: 'stone_block',
      outputMaxStack: 64,
      outputCategory: 'building',
      ingredients: [{
        itemKey: 'rock',
        name: 'Rock',
        iconKey: 'rock',
        maxStack: 64,
        category: 'resource',
        required: 4,
        owned: 5,
      }],
    })

    expect(recipe).toMatchObject({
      outputKey: 'stone_block',
      outputIconKey: 'stone_block',
      outputMaxStack: 64,
      outputCategory: 'building',
    })
    const firstIngredient = recipe.ingredients[0]
    expect(firstIngredient).toBeDefined()
    expect(firstIngredient).toMatchObject({
      itemKey: 'rock',
      iconKey: 'rock',
      maxStack: 64,
      category: 'resource',
    })
  })
})
