import { authJson } from './http'
import type { EquipmentSlot, InventoryItem, StudentInventory } from '../types/inventory'

export interface InventoryItemResponse {
  key?: string
  name?: string
  description?: string
  quantity?: number
  equipSlot?: EquipmentSlot | ''
  visualKey?: string
  iconKey?: string
  maxStack?: number
  category?: string
  equipped?: boolean
}

export interface StudentInventoryResponse {
  items?: InventoryItemResponse[]
}

export async function getStudentInventory(): Promise<StudentInventory> {
  const response = await authJson<StudentInventoryResponse>('/api/student/inventory')
  return {
    items: (response.items || []).map(normalizeInventoryItem).filter((item) => item.quantity > 0),
  }
}

export function normalizeInventoryItem(item: InventoryItemResponse): InventoryItem {
  return {
    key: item.key || '',
    name: item.name || '',
    description: item.description || '',
    quantity: item.quantity ?? 0,
    equipSlot: item.equipSlot || '',
    visualKey: item.visualKey || '',
    iconKey: item.iconKey || item.key || '',
    maxStack: item.maxStack ?? 0,
    category: item.category || '',
    equipped: item.equipped ?? false,
  }
}
