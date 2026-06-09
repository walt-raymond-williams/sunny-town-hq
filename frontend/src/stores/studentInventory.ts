import { defineStore } from 'pinia'
import { craftStudentRecipe, getCraftingRecipes } from '../api/craftingApi'
import { equipStudentItem, getStudentEquipment, unequipStudentItem } from '../api/equipmentApi'
import { getStudentHotbar, setStudentHotbarSlot } from '../api/hotbarApi'
import { getStudentInventory } from '../api/inventoryApi'
import { withKnownCraftingRecipes } from './craftingRecipes'
import type { CraftingRecipe, EquippedSlot, EquipmentSlot, HotbarSlot, InventoryItem, StudentHotbar, StudentInventory } from '../types/inventory'

interface StudentInventoryState {
  items: InventoryItem[]
  craftingRecipes: CraftingRecipe[]
  equipmentSlots: EquippedSlot[]
  hotbarSlots: HotbarSlot[]
  isLoading: boolean
  isUpdatingHotbar: boolean
  isLoadingCrafting: boolean
  isCrafting: boolean
  isUpdatingEquipment: boolean
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
    craftingRecipes: [],
    equipmentSlots: defaultEquipmentSlots,
    hotbarSlots: defaultHotbarSlots,
    isLoading: false,
    isUpdatingHotbar: false,
    isLoadingCrafting: false,
    isCrafting: false,
    isUpdatingEquipment: false,
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
      this.markEquippedItems()
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
      this.equipmentSlots = mergeEquipmentSlots(this.equipmentSlots)
      this.hotbarSlots = mergeHotbarSlots(hotbar.slots)
      this.markEquippedItems()
      this.syncHotbarQuantities()
      this.error = ''
    },
    setItemQuantity(itemKey: string, quantity: number) {
      if (quantity <= 0) {
        this.items = this.items.filter((item) => item.key !== itemKey)
        this.markEquippedItems()
        this.syncHotbarQuantities()
        return
      }
      const existing = this.items.find((item) => item.key === itemKey)
      if (existing) {
        this.items = this.items.map((item) => (
          item.key === itemKey ? { ...item, quantity } : item
        ))
        this.markEquippedItems()
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
      this.markEquippedItems()
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
          getStudentInventory(),
          getStudentEquipment(),
        ])
        this.items = inventory.items
        this.equipmentSlots = mergeEquipmentSlots(equipment.slots)
        this.markEquippedItems()
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
        this.craftingRecipes = withKnownCraftingRecipes(result.recipes, this.items)
        this.markEquippedItems()
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
