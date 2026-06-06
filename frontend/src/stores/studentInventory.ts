import { defineStore } from 'pinia'
import { craftStudentRecipe, getCraftingRecipes } from '../api/craftingApi'
import { equipStudentItem, getStudentEquipment, unequipStudentItem } from '../api/equipmentApi'
import { getStudentInventory } from '../api/inventoryApi'
import type { CraftingRecipe, EquippedSlot, EquipmentSlot, InventoryItem } from '../types/inventory'

interface StudentInventoryState {
  items: InventoryItem[]
  craftingRecipes: CraftingRecipe[]
  equipmentSlots: EquippedSlot[]
  isLoading: boolean
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

export const useStudentInventoryStore = defineStore('studentInventory', {
  state: (): StudentInventoryState => ({
    items: [],
    craftingRecipes: [],
    equipmentSlots: defaultEquipmentSlots,
    isLoading: false,
    isLoadingCrafting: false,
    isCrafting: false,
    isUpdatingEquipment: false,
    error: '',
    craftingError: '',
  }),
  getters: {
    cookieQuantity: (state) => state.items.find((item) => item.key === 'cookie')?.quantity ?? 0,
    unequippedItems: (state) => state.items.filter((item) => !item.equipped),
    craftableRecipes: (state) => state.craftingRecipes.filter((recipe) => recipe.canCraft),
    equippedVisuals: (state) => Object.fromEntries(
      state.equipmentSlots.map((slot) => [slot.slot, slot.item?.visualKey || '']),
    ) as Record<EquipmentSlot, string>,
  },
  actions: {
    setItems(items: InventoryItem[]) {
      this.items = items
      this.markEquippedItems()
      this.error = ''
    },
    setItemQuantity(itemKey: string, quantity: number) {
      if (quantity <= 0) {
        this.items = this.items.filter((item) => item.key !== itemKey)
        this.markEquippedItems()
        return
      }
      const existing = this.items.find((item) => item.key === itemKey)
      if (existing) {
        this.items = this.items.map((item) => (
          item.key === itemKey ? { ...item, quantity } : item
        ))
        this.markEquippedItems()
        return
      }
      const names: Record<string, { name: string; description: string }> = {
        rock: { name: 'Rock', description: 'A sturdy rock from Forest Crossing.' },
        crystal: { name: 'Crystal', description: 'A bright crystal from Forest Crossing.' },
        stone_block: { name: 'Stone Block', description: 'A solid block crafted from stone.' },
      }
      const fallback = names[itemKey] || { name: itemKey, description: '' }
      this.items = [
        ...this.items,
        {
          key: itemKey,
          name: fallback.name,
          description: fallback.description,
          quantity,
          equipSlot: '',
          visualKey: '',
          equipped: false,
        },
      ]
      this.markEquippedItems()
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
      } catch (error) {
        this.error = error instanceof Error ? error.message : String(error)
      } finally {
        this.isLoading = false
      }
    },
    async loadCraftingRecipes() {
      this.isLoadingCrafting = true
      this.craftingError = ''

      try {
        this.craftingRecipes = await getCraftingRecipes()
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
        this.craftingRecipes = result.recipes
        this.markEquippedItems()
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
  },
})

function mergeEquipmentSlots(slots: EquippedSlot[]): EquippedSlot[] {
  return defaultEquipmentSlots.map((defaultSlot) => (
    slots.find((slot) => slot.slot === defaultSlot.slot) || defaultSlot
  ))
}
