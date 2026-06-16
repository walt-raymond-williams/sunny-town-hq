import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { setStudentHotbarSlot } from '../api/hotbarApi'
import { moveStudentInventoryStack } from '../api/inventoryApi'
import type { EquippedSlot, InventoryItem, StudentInventorySlots } from '../types/inventory'
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
    vi.mocked(setStudentHotbarSlot).mockReset()
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

  it('splits a source stack into an empty slot through the API', async () => {
    const store = useStudentInventoryStore()
    store.setInventorySlots(inventoryWithRock(0, 8))
    vi.mocked(moveStudentInventoryStack).mockResolvedValueOnce({
      slotCount: 3,
      slots: [
        { slotIndex: 0, item: item({ quantity: 5 }) },
        { slotIndex: 1, item: null },
        { slotIndex: 2, item: item({ quantity: 3 }) },
      ],
      items: [item({ quantity: 8 })],
    })

    await expect(store.splitInventorySlot(0, 2, 3)).resolves.toBe(true)

    expect(moveStudentInventoryStack).toHaveBeenCalledWith({
      source: { kind: 'player_inventory', slotIndex: 0 },
      destination: { kind: 'player_inventory', slotIndex: 2 },
      mode: 'split',
      quantity: 3,
    })
    expect(store.inventorySlots[0]?.item).toMatchObject({ key: 'rock', quantity: 5 })
    expect(store.inventorySlots[2]?.item).toMatchObject({ key: 'rock', quantity: 3 })
    expect(store.pendingInventoryMoveSourceIndex).toBeNull()
    expect(store.pendingInventoryMoveDestinationIndex).toBeNull()
    expect(store.isMovingInventorySlot).toBe(false)
  })

  it('rejects invalid split requests before calling the API', async () => {
    const store = useStudentInventoryStore()
    store.setInventorySlots(inventoryWithRock(0, 8))

    await expect(store.splitInventorySlot(0, 2, 0)).resolves.toBe(false)
    await expect(store.splitInventorySlot(0, 2, 8)).resolves.toBe(false)
    await expect(store.splitInventorySlot(1, 2, 1)).resolves.toBe(false)

    expect(moveStudentInventoryStack).not.toHaveBeenCalled()
    expect(store.error).toBe('invalid split quantity')
    expect(store.invalidInventoryDropSlotIndex).toBe(1)
  })

  it('assigns a dragged inventory item to a hotbar slot through the hotbar API', async () => {
    const store = useStudentInventoryStore()
    store.setInventorySlots(inventoryWithRock(0))
    vi.mocked(setStudentHotbarSlot).mockResolvedValueOnce({
      slots: [
        { slot: 1, item: item() },
        { slot: 2, item: null },
        { slot: 3, item: null },
        { slot: 4, item: null },
        { slot: 5, item: null },
      ],
    })

    expect(store.startInventorySlotDrag(0)).toBe(true)
    await expect(store.dropInventorySlotOnHotbar(1)).resolves.toBe(true)

    expect(setStudentHotbarSlot).toHaveBeenCalledWith(1, 'rock')
    expect(store.hotbarSlots[0]?.item).toMatchObject({ key: 'rock', quantity: 3 })
    expect(store.draggedInventorySlotIndex).toBeNull()
    expect(store.pendingHotbarDropSlot).toBeNull()
    expect(store.invalidHotbarDropSlot).toBeNull()
  })

  it('rejects hotbar drops without an occupied source before calling the API', async () => {
    const store = useStudentInventoryStore()
    store.setInventorySlots(inventoryWithRock(0))

    await expect(store.dropInventorySlotOnHotbar(2)).resolves.toBe(false)
    expect(store.invalidHotbarDropSlot).toBe(2)

    store.draggedInventorySlotIndex = 1
    await expect(store.dropInventorySlotOnHotbar(3)).resolves.toBe(false)

    expect(setStudentHotbarSlot).not.toHaveBeenCalled()
    expect(store.invalidInventoryDropSlotIndex).toBe(1)
    expect(store.invalidHotbarDropSlot).toBe(3)
    expect(store.draggedInventorySlotIndex).toBeNull()
  })

  it('leaves the last confirmed hotbar state and clears pending state when assignment fails', async () => {
    const store = useStudentInventoryStore()
    store.setInventorySlots(inventoryWithRock(0))
    store.setHotbar({
      slots: [
        { slot: 1, item: item({ key: 'pickaxe', name: 'Pickaxe', quantity: 1 }) },
        { slot: 2, item: null },
        { slot: 3, item: null },
        { slot: 4, item: null },
        { slot: 5, item: null },
      ],
    })
    vi.mocked(setStudentHotbarSlot).mockRejectedValueOnce(new Error('hotbar unavailable'))

    expect(store.startInventorySlotDrag(0)).toBe(true)
    await expect(store.dropInventorySlotOnHotbar(1)).resolves.toBe(false)

    expect(store.hotbarSlots[0]?.item).toMatchObject({ key: 'pickaxe', quantity: 0 })
    expect(store.error).toBe('hotbar unavailable')
    expect(store.pendingHotbarDropSlot).toBeNull()
    expect(store.draggedInventorySlotIndex).toBeNull()
  })

  it('keeps tools assignable to the hotbar instead of the equipment rail', async () => {
    const store = useStudentInventoryStore()
    store.setInventorySlots({
      slotCount: 3,
      slots: [
        { slotIndex: 0, item: item({ key: 'pickaxe', name: 'Pickaxe', quantity: 1, equipSlot: 'tool', maxStack: 1, category: 'tool' }) },
        { slotIndex: 1, item: null },
        { slotIndex: 2, item: null },
      ],
      items: [item({ key: 'pickaxe', name: 'Pickaxe', quantity: 1, equipSlot: 'tool', maxStack: 1, category: 'tool' })],
    })
    vi.mocked(setStudentHotbarSlot).mockResolvedValueOnce({
      slots: [
        { slot: 1, item: item({ key: 'pickaxe', name: 'Pickaxe', quantity: 1, equipSlot: 'tool', maxStack: 1, category: 'tool' }) },
        { slot: 2, item: null },
        { slot: 3, item: null },
        { slot: 4, item: null },
        { slot: 5, item: null },
      ],
    })

    expect(store.startInventorySlotDrag(0)).toBe(true)
    await expect(store.dropInventorySlotOnHotbar(1)).resolves.toBe(true)

    expect(setStudentHotbarSlot).toHaveBeenCalledWith(1, 'pickaxe')
    expect(store.hotbarSlots[0]?.item).toMatchObject({ key: 'pickaxe', quantity: 1 })
    expect(store.pendingHotbarDropSlot).toBeNull()
    expect(store.draggedInventorySlotIndex).toBeNull()
  })

  it('rejects incompatible equipment drops before calling the equipment handler', async () => {
    const store = useStudentInventoryStore()
    store.setInventorySlots({
      slotCount: 3,
      slots: [
        { slotIndex: 0, item: item({ key: 'rock', equipSlot: '', category: 'resource' }) },
        { slotIndex: 1, item: item({ key: 'pickaxe', name: 'Pickaxe', quantity: 1, equipSlot: 'tool', maxStack: 1, category: 'tool' }) },
        { slotIndex: 2, item: null },
      ],
      items: [
        item({ key: 'rock', equipSlot: '', category: 'resource' }),
        item({ key: 'pickaxe', name: 'Pickaxe', quantity: 1, equipSlot: 'tool', maxStack: 1, category: 'tool' }),
      ],
    })
    const equip = vi.fn().mockResolvedValue(undefined)

    expect(store.startInventorySlotDrag(0)).toBe(true)
    await expect(store.dropInventorySlotOnEquipment('gear', equip)).resolves.toBe(false)
    expect(equip).not.toHaveBeenCalled()
    expect(store.invalidEquipmentDropSlot).toBe('gear')

    expect(store.startInventorySlotDrag(1)).toBe(true)
    await expect(store.dropInventorySlotOnEquipment('gear', equip)).resolves.toBe(false)
    expect(equip).not.toHaveBeenCalled()
    expect(store.invalidEquipmentDropSlot).toBe('gear')
  })

  it('filters legacy tool equipment out of the visible equipment slots', () => {
    const store = useStudentInventoryStore()
    const slots: EquippedSlot[] = [
      { slot: 'gear', item: null },
      { slot: 'accessory', item: null },
      {
        slot: 'tool',
        item: {
          key: 'pickaxe',
          name: 'Pickaxe',
          description: 'A sturdy pickaxe.',
          equipSlot: 'tool',
          visualKey: 'pickaxe',
          iconKey: 'pickaxe',
          maxStack: 1,
          category: 'tool',
        },
      },
    ]

    store.setEquipmentSlots(slots)

    expect(store.equipmentSlots.map((slot) => slot.slot)).toEqual(['gear', 'accessory'])
    expect(store.equippedVisuals).toEqual({ gear: '', accessory: '' })
  })
})
