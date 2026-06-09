import { defineStore } from 'pinia'
import { craftStudentRecipe, getCraftingRecipes } from '../api/craftingApi'
import { equipStudentItem, getStudentEquipment, unequipStudentItem } from '../api/equipmentApi'
import { getStudentHotbar, setStudentHotbarSlot } from '../api/hotbarApi'
import { getStudentInventorySlots, moveStudentInventoryStack } from '../api/inventoryApi'
import { withKnownCraftingRecipes } from './craftingRecipes'
import type { CraftingRecipe, EquippedSlot, EquipmentSlot, HotbarSlot, InventoryItem, InventorySlot, StudentHotbar, StudentInventory, StudentInventorySlots } from '../types/inventory'

interface StudentInventoryState {
  items: InventoryItem[]
  inventorySlotCount: number
  inventorySlots: InventorySlot[]
  craftingRecipes: CraftingRecipe[]
  equipmentSlots: EquippedSlot[]
  hotbarSlots: HotbarSlot[]
  isLoading: boolean
  isUpdatingHotbar: boolean
  isLoadingCrafting: boolean
  isCrafting: boolean
  isUpdatingEquipment: boolean
  isMovingInventorySlot: boolean
  draggedInventorySlotIndex: number | null
  pendingInventoryMoveSourceIndex: number | null
  pendingInventoryMoveDestinationIndex: number | null
  invalidInventoryDropSlotIndex: number | null
  pendingHotbarDropSlot: number | null
  invalidHotbarDropSlot: number | null
  pendingEquipmentDropSlot: EquipmentSlot | null
  invalidEquipmentDropSlot: EquipmentSlot | null
  error: string
  craftingError: string
}

const defaultEquipmentSlots: EquippedSlot[] = [
  { slot: 'gear', item: null },
  { slot: 'accessory', item: null },
  { slot: 'tool', item: null },
]

const defaultHotbarSlots: HotbarSlot[] = Array.from({ length: 5 }, (_, index) => ({
  slot: index + 1,
  item: null,
}))

export const useStudentInventoryStore = defineStore('studentInventory', {
  state: (): StudentInventoryState => ({
    items: [],
    inventorySlotCount: 0,
    inventorySlots: [],
    craftingRecipes: [],
    equipmentSlots: defaultEquipmentSlots,
    hotbarSlots: defaultHotbarSlots,
    isLoading: false,
    isUpdatingHotbar: false,
    isLoadingCrafting: false,
    isCrafting: false,
    isUpdatingEquipment: false,
    isMovingInventorySlot: false,
    draggedInventorySlotIndex: null,
    pendingInventoryMoveSourceIndex: null,
    pendingInventoryMoveDestinationIndex: null,
    invalidInventoryDropSlotIndex: null,
    pendingHotbarDropSlot: null,
    invalidHotbarDropSlot: null,
    pendingEquipmentDropSlot: null,
    invalidEquipmentDropSlot: null,
    error: '',
    craftingError: '',
  }),
  getters: {
    cookieQuantity: (state) => state.items.find((item) => item.key === 'cookie')?.quantity ?? 0,
    unequippedItems: (state) => state.items.filter((item) => !item.equipped),
    knownCraftingRecipes: (state) => withKnownCraftingRecipes(state.craftingRecipes, state.items),
    craftableRecipes: (state) => withKnownCraftingRecipes(state.craftingRecipes, state.items).filter((recipe) => recipe.canCraft),
    equippedVisuals: (state) => Object.fromEntries(
      state.equipmentSlots.map((slot) => [slot.slot, slot.item?.visualKey || '']),
    ) as Record<EquipmentSlot, string>,
  },
  actions: {
    setItems(items: InventoryItem[]) {
      this.items = items
      this.inventorySlots = slotsFromItems(items, this.inventorySlotCount || 30)
      this.inventorySlotCount = this.inventorySlots.length
      this.markEquippedItems()
      this.syncHotbarQuantities()
      this.error = ''
    },
    setInventorySlots(inventory: StudentInventorySlots) {
      this.items = inventory.items
      this.inventorySlotCount = inventory.slotCount
      this.inventorySlots = inventory.slots
      this.markEquippedItems()
      this.syncSlotEquippedFlags()
      this.syncHotbarQuantities()
      this.error = ''
    },
    setHotbar(hotbar: StudentHotbar) {
      this.hotbarSlots = mergeHotbarSlots(hotbar.slots)
      this.syncHotbarQuantities()
      this.error = ''
    },
    setSessionInventory(inventory: StudentInventory, hotbar: StudentHotbar) {
      this.items = inventory.items
      this.inventorySlots = slotsFromItems(inventory.items, this.inventorySlotCount || 30)
      this.inventorySlotCount = this.inventorySlots.length
      this.equipmentSlots = mergeEquipmentSlots(this.equipmentSlots)
      this.hotbarSlots = mergeHotbarSlots(hotbar.slots)
      this.markEquippedItems()
      this.syncHotbarQuantities()
      this.error = ''
    },
    setItemQuantity(itemKey: string, quantity: number) {
      if (quantity <= 0) {
        this.items = this.items.filter((item) => item.key !== itemKey)
        this.inventorySlots = this.inventorySlots.map((slot) => (
          slot.item?.key === itemKey ? { ...slot, item: null } : slot
        ))
        this.markEquippedItems()
        this.syncSlotEquippedFlags()
        this.syncHotbarQuantities()
        return
      }
      const existing = this.items.find((item) => item.key === itemKey)
      if (existing) {
        this.items = this.items.map((item) => (
          item.key === itemKey ? { ...item, quantity } : item
        ))
        this.syncItemSlots(itemKey, quantity, existing)
        this.markEquippedItems()
        this.syncSlotEquippedFlags()
        this.syncHotbarQuantities()
        return
      }
      const names: Record<string, { name: string; description: string }> = {
        rock: { name: 'Rock', description: 'A sturdy rock from Forest Crossing.' },
        crystal: { name: 'Crystal', description: 'A bright crystal from Forest Crossing.' },
        stone_block: { name: 'Stone Block', description: 'A solid block crafted from stone.' },
      }
      const fallback = names[itemKey] || { name: itemKey, description: '' }
      const metadata = fallbackItemMetadata(itemKey)
      this.items = [
        ...this.items,
        {
          key: itemKey,
          name: fallback.name,
          description: fallback.description,
          quantity,
          equipSlot: '',
          visualKey: '',
          iconKey: metadata.iconKey,
          maxStack: metadata.maxStack,
          category: metadata.category,
          equipped: false,
        },
      ]
      this.syncItemSlots(itemKey, quantity, this.items.find((item) => item.key === itemKey) || null)
      this.markEquippedItems()
      this.syncSlotEquippedFlags()
      this.syncHotbarQuantities()
    },
    setEquipmentSlots(slots: EquippedSlot[]) {
      this.equipmentSlots = mergeEquipmentSlots(slots)
      this.markEquippedItems()
      this.error = ''
    },
    async loadInventory() {
      this.isLoading = true
      this.error = ''

      try {
        const [inventory, equipment] = await Promise.all([
          getStudentInventorySlots(),
          getStudentEquipment(),
        ])
        this.items = inventory.items
        this.inventorySlotCount = inventory.slotCount
        this.inventorySlots = inventory.slots
        this.equipmentSlots = mergeEquipmentSlots(equipment.slots)
        this.markEquippedItems()
        this.syncSlotEquippedFlags()
        this.syncHotbarQuantities()
      } catch (error) {
        this.error = error instanceof Error ? error.message : String(error)
      } finally {
        this.isLoading = false
      }
    },
    async loadHotbar() {
      this.isUpdatingHotbar = true
      this.error = ''

      try {
        const hotbar = await getStudentHotbar()
        this.hotbarSlots = mergeHotbarSlots(hotbar.slots)
        this.syncHotbarQuantities()
      } catch (error) {
        this.error = error instanceof Error ? error.message : String(error)
      } finally {
        this.isUpdatingHotbar = false
      }
    },
    async setHotbarSlot(slot: number, itemKey: string) {
      this.isUpdatingHotbar = true
      this.error = ''

      try {
        const hotbar = await setStudentHotbarSlot(slot, itemKey)
        this.hotbarSlots = mergeHotbarSlots(hotbar.slots)
        this.syncHotbarQuantities()
      } catch (error) {
        this.error = error instanceof Error ? error.message : String(error)
        throw error
      } finally {
        this.isUpdatingHotbar = false
      }
    },
    startInventorySlotDrag(slotIndex: number): boolean {
      if (this.isMovingInventorySlot) {
        return false
      }
      const slot = this.inventorySlots.find((candidate) => candidate.slotIndex === slotIndex)
      if (!slot?.item) {
        this.draggedInventorySlotIndex = null
        this.invalidInventoryDropSlotIndex = slotIndex
        return false
      }
      this.draggedInventorySlotIndex = slotIndex
      this.invalidInventoryDropSlotIndex = null
      this.invalidHotbarDropSlot = null
      this.invalidEquipmentDropSlot = null
      this.error = ''
      return true
    },
    setInventorySlotDropTarget(slotIndex: number) {
      if (this.draggedInventorySlotIndex === null) {
        this.invalidInventoryDropSlotIndex = slotIndex
        return
      }
      this.invalidInventoryDropSlotIndex = this.draggedInventorySlotIndex === slotIndex ? slotIndex : null
    },
    clearInventorySlotDropTarget(slotIndex: number) {
      if (this.invalidInventoryDropSlotIndex === slotIndex) {
        this.invalidInventoryDropSlotIndex = null
      }
    },
    setHotbarDropTarget(slot: number) {
      if (this.draggedInventorySlotIndex === null) {
        this.invalidHotbarDropSlot = slot
        return
      }
      this.invalidHotbarDropSlot = null
    },
    clearHotbarDropTarget(slot: number) {
      if (this.invalidHotbarDropSlot === slot) {
        this.invalidHotbarDropSlot = null
      }
    },
    setEquipmentDropTarget(slot: EquipmentSlot) {
      if (this.draggedInventorySlotIndex === null) {
        this.invalidEquipmentDropSlot = slot
        return
      }
      const sourceSlot = this.inventorySlots.find((candidate) => candidate.slotIndex === this.draggedInventorySlotIndex)
      const sourceItem = sourceSlot?.item
      this.invalidEquipmentDropSlot = sourceItem?.equipSlot === slot ? null : slot
    },
    clearEquipmentDropTarget(slot: EquipmentSlot) {
      if (this.invalidEquipmentDropSlot === slot) {
        this.invalidEquipmentDropSlot = null
      }
    },
    cancelInventorySlotDrag() {
      if (!this.isMovingInventorySlot) {
        this.draggedInventorySlotIndex = null
        this.invalidInventoryDropSlotIndex = null
        this.invalidHotbarDropSlot = null
        this.invalidEquipmentDropSlot = null
      }
    },
    async dropInventorySlot(slotIndex: number): Promise<boolean> {
      const sourceSlotIndex = this.draggedInventorySlotIndex
      if (sourceSlotIndex === null) {
        this.invalidInventoryDropSlotIndex = slotIndex
        return false
      }
      try {
        return await this.moveInventorySlot(sourceSlotIndex, slotIndex)
      } finally {
        this.draggedInventorySlotIndex = null
        this.invalidInventoryDropSlotIndex = null
        this.invalidHotbarDropSlot = null
        this.invalidEquipmentDropSlot = null
      }
    },
    async dropInventorySlotOnHotbar(slot: number): Promise<boolean> {
      const sourceSlotIndex = this.draggedInventorySlotIndex
      if (sourceSlotIndex === null) {
        this.invalidHotbarDropSlot = slot
        return false
      }
      const sourceSlot = this.inventorySlots.find((candidate) => candidate.slotIndex === sourceSlotIndex)
      if (!sourceSlot?.item) {
        this.invalidInventoryDropSlotIndex = sourceSlotIndex
        this.invalidHotbarDropSlot = slot
        this.draggedInventorySlotIndex = null
        return false
      }

      this.pendingHotbarDropSlot = slot
      this.invalidHotbarDropSlot = null
      this.error = ''

      try {
        await this.setHotbarSlot(slot, sourceSlot.item.key)
        return true
      } catch {
        return false
      } finally {
        this.pendingHotbarDropSlot = null
        this.draggedInventorySlotIndex = null
        this.invalidInventoryDropSlotIndex = null
      }
    },
    async dropInventorySlotOnEquipment(
      slot: EquipmentSlot,
      equip?: (slot: EquipmentSlot, itemKey: string) => Promise<void>,
    ): Promise<boolean> {
      const sourceSlotIndex = this.draggedInventorySlotIndex
      if (sourceSlotIndex === null) {
        this.invalidEquipmentDropSlot = slot
        return false
      }
      const sourceSlot = this.inventorySlots.find((candidate) => candidate.slotIndex === sourceSlotIndex)
      if (!sourceSlot?.item) {
        this.invalidInventoryDropSlotIndex = sourceSlotIndex
        this.invalidEquipmentDropSlot = slot
        this.draggedInventorySlotIndex = null
        return false
      }
      if (sourceSlot.item.equipSlot !== slot) {
        this.invalidEquipmentDropSlot = slot
        this.draggedInventorySlotIndex = null
        return false
      }

      this.pendingEquipmentDropSlot = slot
      this.invalidEquipmentDropSlot = null
      this.error = ''

      try {
        if (equip) {
          await equip(slot, sourceSlot.item.key)
        } else {
          await this.equipItem(slot, sourceSlot.item.key)
        }
        return true
      } catch {
        return false
      } finally {
        this.pendingEquipmentDropSlot = null
        this.draggedInventorySlotIndex = null
        this.invalidInventoryDropSlotIndex = null
      }
    },
    async moveInventorySlot(sourceSlotIndex: number, destinationSlotIndex: number): Promise<boolean> {
      if (this.isMovingInventorySlot) {
        return false
      }
      const sourceSlot = this.inventorySlots.find((slot) => slot.slotIndex === sourceSlotIndex)
      if (!sourceSlot?.item) {
        this.invalidInventoryDropSlotIndex = sourceSlotIndex
        return false
      }
      if (sourceSlotIndex === destinationSlotIndex) {
        this.invalidInventoryDropSlotIndex = destinationSlotIndex
        return false
      }

      this.isMovingInventorySlot = true
      this.pendingInventoryMoveSourceIndex = sourceSlotIndex
      this.pendingInventoryMoveDestinationIndex = destinationSlotIndex
      this.invalidInventoryDropSlotIndex = null
      this.error = ''

      try {
        const inventory = await moveStudentInventoryStack({
          source: { kind: 'player_inventory', slotIndex: sourceSlotIndex },
          destination: { kind: 'player_inventory', slotIndex: destinationSlotIndex },
          mode: 'auto',
        })
        this.setInventorySlots(inventory)
        return true
      } catch (error) {
        this.error = error instanceof Error ? error.message : String(error)
        return false
      } finally {
        this.isMovingInventorySlot = false
        this.pendingInventoryMoveSourceIndex = null
        this.pendingInventoryMoveDestinationIndex = null
      }
    },
    async loadCraftingRecipes() {
      this.isLoadingCrafting = true
      this.craftingError = ''

      try {
        this.craftingRecipes = withKnownCraftingRecipes(await getCraftingRecipes(), this.items)
      } catch (error) {
        this.craftingError = error instanceof Error ? error.message : String(error)
      } finally {
        this.isLoadingCrafting = false
      }
    },
    async craftRecipe(recipeKey: string) {
      this.isCrafting = true
      this.craftingError = ''

      try {
        const result = await craftStudentRecipe(recipeKey)
        this.items = result.inventory.items
        this.inventorySlots = slotsFromItems(this.items, this.inventorySlotCount || 30)
        this.inventorySlotCount = this.inventorySlots.length
        this.craftingRecipes = withKnownCraftingRecipes(result.recipes, this.items)
        this.markEquippedItems()
        this.syncSlotEquippedFlags()
        this.syncHotbarQuantities()
      } catch (error) {
        this.craftingError = error instanceof Error ? error.message : String(error)
        throw error
      } finally {
        this.isCrafting = false
      }
    },
    async equipItem(slot: EquipmentSlot, itemKey: string) {
      this.isUpdatingEquipment = true
      this.error = ''

      try {
        const equipment = await equipStudentItem(slot, itemKey)
        this.equipmentSlots = mergeEquipmentSlots(equipment.slots)
        await this.loadInventory()
      } catch (error) {
        this.error = error instanceof Error ? error.message : String(error)
        throw error
      } finally {
        this.isUpdatingEquipment = false
      }
    },
    async unequipItem(slot: EquipmentSlot) {
      this.isUpdatingEquipment = true
      this.error = ''

      try {
        const equipment = await unequipStudentItem(slot)
        this.equipmentSlots = mergeEquipmentSlots(equipment.slots)
        await this.loadInventory()
      } catch (error) {
        this.error = error instanceof Error ? error.message : String(error)
        throw error
      } finally {
        this.isUpdatingEquipment = false
      }
    },
    markEquippedItems() {
      const equippedKeys = new Set(
        this.equipmentSlots
          .map((slot) => slot.item?.key || '')
          .filter(Boolean),
      )
      this.items = this.items.map((item) => ({
        ...item,
        equipped: equippedKeys.has(item.key),
      }))
    },
    syncSlotEquippedFlags() {
      const byKey = new Map(this.items.map((item) => [item.key, item]))
      this.inventorySlots = this.inventorySlots.map((slot) => ({
        ...slot,
        item: slot.item ? { ...slot.item, equipped: byKey.get(slot.item.key)?.equipped ?? slot.item.equipped } : null,
      }))
    },
    syncItemSlots(itemKey: string, quantity: number, metadataSource: InventoryItem | null) {
      const slotCount = this.inventorySlotCount || 30
      let remaining = quantity
      let metadata = metadataSource || this.items.find((item) => item.key === itemKey) || null
      if (!metadata) {
        const fallback = fallbackItemMetadata(itemKey)
        metadata = {
          key: itemKey,
          name: itemKey,
          description: '',
          quantity: 0,
          equipSlot: '',
          visualKey: '',
          iconKey: fallback.iconKey,
          maxStack: fallback.maxStack,
          category: fallback.category,
          equipped: false,
        }
      }
      const maxStack = metadata.maxStack > 0 ? metadata.maxStack : quantity
      const nextSlots = ensureSlotCount(this.inventorySlots, slotCount).map((slot) => {
        if (slot.item?.key !== itemKey) {
          return slot
        }
        if (remaining <= 0) {
          return { ...slot, item: null }
        }
        const slotQuantity = Math.min(remaining, maxStack)
        remaining -= slotQuantity
        return {
          ...slot,
          item: { ...metadata, quantity: slotQuantity },
        }
      })
      for (const slot of nextSlots) {
        if (remaining <= 0) {
          break
        }
        if (slot.item) {
          continue
        }
        const slotQuantity = Math.min(remaining, maxStack)
        slot.item = { ...metadata, quantity: slotQuantity }
        remaining -= slotQuantity
      }
      this.inventorySlots = nextSlots
      this.inventorySlotCount = nextSlots.length
    },
    syncHotbarQuantities() {
      const byKey = new Map(this.items.map((item) => [item.key, item]))
      this.hotbarSlots = mergeHotbarSlots(this.hotbarSlots).map((slot) => {
        if (!slot.item) {
          return slot
        }
        const inventoryItem = byKey.get(slot.item.key)
        return {
          ...slot,
          item: inventoryItem ? { ...inventoryItem } : { ...slot.item, quantity: 0 },
        }
      })
    },
  },
})

function mergeEquipmentSlots(slots: EquippedSlot[]): EquippedSlot[] {
  return defaultEquipmentSlots.map((defaultSlot) => (
    slots.find((slot) => slot.slot === defaultSlot.slot) || defaultSlot
  ))
}

function mergeHotbarSlots(slots: HotbarSlot[]): HotbarSlot[] {
  return defaultHotbarSlots.map((defaultSlot) => {
    const slot = slots.find((candidate) => candidate.slot === defaultSlot.slot)
    return slot ? { slot: slot.slot, item: slot.item ? { ...slot.item } : null } : { ...defaultSlot }
  })
}

function slotsFromItems(items: InventoryItem[], slotCount: number): InventorySlot[] {
  const slots = ensureSlotCount([], slotCount)
  let slotIndex = 0
  for (const item of items) {
    let remaining = item.quantity
    const maxStack = item.maxStack > 0 ? item.maxStack : item.quantity
    while (remaining > 0 && slotIndex < slots.length) {
      const quantity = Math.min(remaining, maxStack)
      slots[slotIndex] = {
        slotIndex,
        item: { ...item, quantity },
      }
      remaining -= quantity
      slotIndex += 1
    }
  }
  return slots
}

function ensureSlotCount(slots: InventorySlot[], slotCount: number): InventorySlot[] {
  const count = Math.max(slotCount, slots.length, 0)
  return Array.from({ length: count }, (_, slotIndex) => {
    const slot = slots.find((candidate) => candidate.slotIndex === slotIndex)
    return slot ? { slotIndex, item: slot.item ? { ...slot.item } : null } : { slotIndex, item: null }
  })
}

function fallbackItemMetadata(itemKey: string): Pick<InventoryItem, 'iconKey' | 'maxStack' | 'category'> {
  const categories: Record<string, string> = {
    cookie: 'consumable',
    sunny_hoodie: 'gear',
    star_cap: 'gear',
    pickaxe: 'tool',
    rock: 'resource',
    crystal: 'resource',
    stone_block: 'building',
  }
  return {
    iconKey: itemKey,
    maxStack: itemKey === 'sunny_hoodie' || itemKey === 'star_cap' || itemKey === 'pickaxe' ? 1 : 64,
    category: categories[itemKey] || '',
  }
}
