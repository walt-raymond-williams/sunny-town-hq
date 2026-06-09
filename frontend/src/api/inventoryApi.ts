import { authJson, jsonOptions } from './http'
import type { EquipmentSlot, InventoryItem, InventoryMoveRequest, InventorySlot, StudentInventory, StudentInventorySlots } from '../types/inventory'

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

export interface InventorySlotResponse {
  slotIndex?: number
  item?: InventoryItemResponse | null
}

export interface StudentInventorySlotsResponse {
  slotCount?: number
  slots?: InventorySlotResponse[]
  items?: InventoryItemResponse[]
}

export async function getStudentInventory(): Promise<StudentInventory> {
  const response = await authJson<StudentInventoryResponse>('/api/student/inventory')
  return {
    items: (response.items || []).map(normalizeInventoryItem).filter((item) => item.quantity > 0),
  }
}

export async function getStudentInventorySlots(): Promise<StudentInventorySlots> {
  const response = await authJson<StudentInventorySlotsResponse>('/api/student/inventory/slots')
  return normalizeStudentInventorySlots(response)
}

export async function moveStudentInventoryStack(request: InventoryMoveRequest): Promise<StudentInventorySlots> {
  const response = await authJson<StudentInventorySlotsResponse>(
    '/api/student/inventory/move',
    jsonOptions('POST', request),
  )
  return normalizeStudentInventorySlots(response)
}

function normalizeStudentInventorySlots(response: StudentInventorySlotsResponse): StudentInventorySlots {
  const slotCount = response.slotCount ?? 0
  const slots = normalizeInventorySlots(response.slots || [], slotCount)
  return {
    slotCount,
    slots,
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

function normalizeInventorySlots(slots: InventorySlotResponse[], slotCount: number): InventorySlot[] {
  const byIndex = new Map(slots.map((slot) => [slot.slotIndex ?? -1, slot]))
  return Array.from({ length: slotCount }, (_, slotIndex) => {
    const slot = byIndex.get(slotIndex)
    return {
      slotIndex,
      item: slot?.item ? normalizeInventoryItem(slot.item) : null,
    }
  })
}
