export interface InventoryItem {
  key: string
  name: string
  description: string
  quantity: number
}

export interface StudentInventory {
  items: InventoryItem[]
}
