import { defineStore } from 'pinia'
import { equipStudentItem, getStudentEquipment, unequipStudentItem } from '../api/equipmentApi'
import { getStudentInventory } from '../api/inventoryApi'
import type { EquippedSlot, EquipmentSlot, InventoryItem } from '../types/inventory'

interface StudentInventoryState {
  items: InventoryItem[]
  equipmentSlots: EquippedSlot[]
  isLoading: boolean
  isUpdatingEquipment: boolean
  error: string
}

const defaultEquipmentSlots: EquippedSlot[] = [
  { slot: 'gear', item: null },
  { slot: 'accessory', item: null },
]

export const useStudentInventoryStore = defineStore('studentInventory', {
  state: (): StudentInventoryState => ({
    items: [],
    equipmentSlots: defaultEquipmentSlots,
    isLoading: false,
    isUpdatingEquipment: false,
    error: '',
  }),
  getters: {
    cookieQuantity: (state) => state.items.find((item) => item.key === 'cookie')?.quantity ?? 0,
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
