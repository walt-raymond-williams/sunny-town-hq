import { defineStore } from 'pinia'
import { getStudentInventory } from '../api/inventoryApi'
import type { InventoryItem } from '../types/inventory'

interface StudentInventoryState {
  items: InventoryItem[]
  isLoading: boolean
  error: string
}

export const useStudentInventoryStore = defineStore('studentInventory', {
  state: (): StudentInventoryState => ({
    items: [],
    isLoading: false,
    error: '',
  }),
  getters: {
    cookieQuantity: (state) => state.items.find((item) => item.key === 'cookie')?.quantity ?? 0,
  },
  actions: {
    async loadInventory() {
      this.isLoading = true
      this.error = ''

      try {
        const inventory = await getStudentInventory()
        this.items = inventory.items
      } catch (error) {
        this.error = error instanceof Error ? error.message : String(error)
      } finally {
        this.isLoading = false
      }
    },
  },
})
