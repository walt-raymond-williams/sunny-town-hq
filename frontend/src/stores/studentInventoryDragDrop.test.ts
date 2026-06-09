import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { moveStudentInventoryStack } from '../api/inventoryApi'
import type { InventoryItem, StudentInventorySlots } from '../types/inventory'
import { useStudentInventoryStore } from './studentInventory'

vi.mock('../api/inventoryApi', () => ({
  getStudentInventorySlots: vi.fn(),
  moveStudentInventoryStack: vi.fn(),
}))

vi.mock('../api/craftingApi', () => ({
  craftStudentRecipe: vi.fn(),
  getCraftingRecipes: vi.fn(),
}))

vi.mock('../api/equipmentApi', () => ({
  equipStudentItem: vi.fn(),
  getStudentEquipment: vi.fn(),
  unequipStudentItem: vi.fn(),
}))

vi.mock('../api/hotbarApi', () => ({
  getStudentHotbar: vi.fn(),
  setStudentHotbarSlot: vi.fn(),
}))

function item(overrides: Partial<InventoryItem> = {}): InventoryItem {
  return {
    key: 'rock',
    name: 'Rock',
    description: 'A sturdy rock.',
    quantity: 3,
    equipSlot: '',
    visualKey: '',
    iconKey: 'rock',
    maxStack: 64,
    category: 'resource',
    equipped: false,
    ...overrides,
  }
}

function inventoryWithRock(slotIndex: number, quantity = 3): StudentInventorySlots {
  const rock = item({ quantity })
  return {
    slotCount: 3,
    slots: Array.from({ length: 3 }, (_, index) => ({
      slotIndex: index,
      item: index === slotIndex ? rock : null,
    })),
    items: [rock],
  }
}

describe('student inventory drag/drop moves', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(moveStudentInventoryStack).mockReset()
  })

  it('rejects dragging or moving from an empty source slot before calling the API', async () => {
    const store = useStudentInventoryStore()
    store.setInventorySlots(inventoryWithRock(0))

    expect(store.startInventorySlotDrag(1)).toBe(false)
    expect(store.draggedInventorySlotIndex).toBeNull()
    expect(store.invalidInventoryDropSlotIndex).toBe(1)

    await expect(store.moveInventorySlot(1, 2)).resolves.toBe(false)
    expect(moveStudentInventoryStack).not.toHaveBeenCalled()
  })

  it('moves a dragged source slot through the API and syncs state from the response', async () => {
    const store = useStudentInventoryStore()
    store.setInventorySlots(inventoryWithRock(0))
    vi.mocked(moveStudentInventoryStack).mockResolvedValueOnce(inventoryWithRock(2))

    expect(store.startInventorySlotDrag(0)).toBe(true)
    const moved = await store.dropInventorySlot(2)

    expect(moved).toBe(true)
    expect(moveStudentInventoryStack).toHaveBeenCalledWith({
      source: { kind: 'player_inventory', slotIndex: 0 },
      destination: { kind: 'player_inventory', slotIndex: 2 },
      mode: 'auto',
    })
    expect(store.inventorySlots[0]?.item).toBeNull()
    expect(store.inventorySlots[2]?.item).toMatchObject({ key: 'rock', quantity: 3 })
    expect(store.draggedInventorySlotIndex).toBeNull()
    expect(store.pendingInventoryMoveSourceIndex).toBeNull()
    expect(store.pendingInventoryMoveDestinationIndex).toBeNull()
    expect(store.isMovingInventorySlot).toBe(false)
  })

  it('leaves the last confirmed slots in place and clears pending state when the API rejects', async () => {
    const store = useStudentInventoryStore()
    store.setInventorySlots(inventoryWithRock(0))
    vi.mocked(moveStudentInventoryStack).mockRejectedValueOnce(new Error('stacks cannot be merged'))

    await expect(store.moveInventorySlot(0, 2)).resolves.toBe(false)

    expect(store.inventorySlots[0]?.item).toMatchObject({ key: 'rock', quantity: 3 })
    expect(store.inventorySlots[2]?.item).toBeNull()
    expect(store.error).toBe('stacks cannot be merged')
    expect(store.pendingInventoryMoveSourceIndex).toBeNull()
    expect(store.pendingInventoryMoveDestinationIndex).toBeNull()
    expect(store.isMovingInventorySlot).toBe(false)
  })

  it('treats same-slot drops as a no-op', async () => {
    const store = useStudentInventoryStore()
    store.setInventorySlots(inventoryWithRock(0))

    expect(await store.moveInventorySlot(0, 0)).toBe(false)

    expect(store.invalidInventoryDropSlotIndex).toBe(0)
    expect(moveStudentInventoryStack).not.toHaveBeenCalled()
  })
})
