export interface InventoryItem {
  key: string
  name: string
  description: string
  quantity: number
  equipSlot: EquipmentSlot | ''
  visualKey: string
  equipped: boolean
}

export interface StudentInventory {
  items: InventoryItem[]
}

export type EquipmentSlot = 'gear' | 'accessory' | 'tool'

export interface EquipmentItem {
  key: string
  name: string
  description: string
  equipSlot: EquipmentSlot
  visualKey: string
}

export interface EquippedSlot {
  slot: EquipmentSlot
  item: EquipmentItem | null
}

export interface StudentEquipment {
  slots: EquippedSlot[]
}
