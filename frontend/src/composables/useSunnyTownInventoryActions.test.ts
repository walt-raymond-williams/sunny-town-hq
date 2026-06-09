import { describe, expect, it, vi } from 'vitest'
import type { HotbarSlot, InventoryItem } from '../types/inventory'
import {
  hotbarIndexForEvent,
  useSunnyTownInventoryActions,
} from './useSunnyTownInventoryActions'

function inventoryItem(overrides: Partial<InventoryItem> = {}): InventoryItem {
  return {
    key: 'stone_block',
    name: 'Stone Block',
    description: 'A solid block.',
    quantity: 2,
    equipSlot: '',
    visualKey: '',
    iconKey: 'stone_block',
    maxStack: 64,
    category: 'building',
    equipped: false,
    ...overrides,
  }
}

function hotbarSlot(slot: number, item: InventoryItem | null): HotbarSlot {
  return { slot, item }
}

interface MockInventoryStore {
  items: InventoryItem[]
  hotbarSlots: HotbarSlot[]
  craftRecipe: ReturnType<typeof vi.fn>
  dropInventorySlotOnEquipment: ReturnType<typeof vi.fn>
  equipItem: ReturnType<typeof vi.fn>
  loadCraftingRecipes: ReturnType<typeof vi.fn>
  loadHotbar: ReturnType<typeof vi.fn>
  loadInventory: ReturnType<typeof vi.fn>
  setHotbarSlot: ReturnType<typeof vi.fn>
  unequipItem: ReturnType<typeof vi.fn>
}

function store(overrides: Partial<MockInventoryStore> = {}): MockInventoryStore {
  return {
    items: [inventoryItem()],
    hotbarSlots: [
      hotbarSlot(1, inventoryItem()),
      hotbarSlot(2, null),
      hotbarSlot(3, null),
      hotbarSlot(4, null),
      hotbarSlot(5, null),
    ],
    craftRecipe: vi.fn(),
    dropInventorySlotOnEquipment: vi.fn(),
    equipItem: vi.fn(),
    loadCraftingRecipes: vi.fn(),
    loadHotbar: vi.fn(),
    loadInventory: vi.fn(),
    setHotbarSlot: vi.fn(),
    unequipItem: vi.fn(),
    ...overrides,
  }
}

describe('hotbarIndexForEvent', () => {
  it('maps digit keys one through five to zero-based hotbar indexes', () => {
    expect(hotbarIndexForEvent({ code: 'Digit1' })).toBe(0)
    expect(hotbarIndexForEvent({ code: 'Digit5' })).toBe(4)
  })

  it('ignores unsupported keyboard events', () => {
    expect(hotbarIndexForEvent({ code: 'Digit0' })).toBeNull()
    expect(hotbarIndexForEvent({ code: 'KeyE' })).toBeNull()
  })
})

describe('useSunnyTownInventoryActions', () => {
  it('opens inventory and loads inventory, hotbar, and crafting data', async () => {
    const inventoryStore = store()
    const actions = useSunnyTownInventoryActions(inventoryStore, {
      onEquipmentChanged: vi.fn(),
      onHotbarSelectionChanged: vi.fn(),
      onHotbarUpdated: vi.fn(),
    })

    await actions.toggleInventory()

    expect(actions.inventoryOpen.value).toBe(true)
    expect(actions.craftingPanelOpen.value).toBe(true)
    expect(actions.showAllCraftingRecipes.value).toBe(true)
    expect(inventoryStore.loadInventory).toHaveBeenCalledOnce()
    expect(inventoryStore.loadHotbar).toHaveBeenCalledOnce()
    expect(inventoryStore.loadCraftingRecipes).toHaveBeenCalledOnce()
  })

  it('updates selected hotbar slot and runs the selection callback', () => {
    const onHotbarSelectionChanged = vi.fn()
    const actions = useSunnyTownInventoryActions(store(), {
      onEquipmentChanged: vi.fn(),
      onHotbarSelectionChanged,
      onHotbarUpdated: vi.fn(),
    })

    actions.selectHotbarSlot(3)

    expect(actions.selectedHotbarIndex.value).toBe(3)
    expect(onHotbarSelectionChanged).toHaveBeenCalledOnce()
  })

  it('assigns and clears the selected hotbar slot using one-based API slots', async () => {
    const inventoryStore = store()
    const onHotbarUpdated = vi.fn()
    const actions = useSunnyTownInventoryActions(inventoryStore, {
      onEquipmentChanged: vi.fn(),
      onHotbarSelectionChanged: vi.fn(),
      onHotbarUpdated,
    })
    actions.selectHotbarSlot(2)

    await actions.assignInventoryItemToSelectedHotbarSlot('rock')
    await actions.clearSelectedHotbarSlot()

    expect(inventoryStore.setHotbarSlot).toHaveBeenNthCalledWith(1, 3, 'rock')
    expect(inventoryStore.setHotbarSlot).toHaveBeenNthCalledWith(2, 3, '')
    expect(onHotbarUpdated).toHaveBeenCalledTimes(2)
  })

  it('updates equipment and skips missing equipment slots', async () => {
    const inventoryStore = store()
    const onEquipmentChanged = vi.fn()
    const actions = useSunnyTownInventoryActions(inventoryStore, {
      onEquipmentChanged,
      onHotbarSelectionChanged: vi.fn(),
      onHotbarUpdated: vi.fn(),
    })

    await actions.equipInventoryItem('pickaxe', '')
    await actions.equipInventoryItem('pickaxe', 'tool')
    await actions.unequipInventorySlot('tool')

    expect(inventoryStore.equipItem).toHaveBeenCalledOnce()
    expect(inventoryStore.equipItem).toHaveBeenCalledWith('tool', 'pickaxe')
    expect(inventoryStore.unequipItem).toHaveBeenCalledWith('tool')
    expect(onEquipmentChanged).toHaveBeenCalledTimes(2)
  })

  it('equips dropped inventory slots through the store and runs the equipment callback from the drop handler', async () => {
    const inventoryStore = store()
    const onEquipmentChanged = vi.fn()
    inventoryStore.dropInventorySlotOnEquipment.mockImplementation(async (slot, equip) => {
      await equip(slot, 'pickaxe')
      return true
    })
    const actions = useSunnyTownInventoryActions(inventoryStore, {
      onEquipmentChanged,
      onHotbarSelectionChanged: vi.fn(),
      onHotbarUpdated: vi.fn(),
    })

    await actions.equipInventorySlotDrop('tool')

    expect(inventoryStore.dropInventorySlotOnEquipment).toHaveBeenCalledOnce()
    expect(inventoryStore.equipItem).toHaveBeenCalledWith('tool', 'pickaxe')
    expect(onEquipmentChanged).toHaveBeenCalledOnce()
  })
})
