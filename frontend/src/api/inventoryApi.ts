import { authJson } from './http'
import type { InventoryItem, StudentInventory } from '../types/inventory'

interface InventoryItemResponse {
  key?: string
  name?: string
  description?: string
  quantity?: number
}

interface StudentInventoryResponse {
  items?: InventoryItemResponse[]
}

export async function getStudentInventory(): Promise<StudentInventory> {
  const response = await authJson<StudentInventoryResponse>('/api/student/inventory')
  return {
    items: (response.items || []).map(normalizeInventoryItem).filter((item) => item.quantity > 0),
  }
}

function normalizeInventoryItem(item: InventoryItemResponse): InventoryItem {
  return {
    key: item.key || '',
    name: item.name || '',
    description: item.description || '',
    quantity: item.quantity ?? 0,
  }
}
