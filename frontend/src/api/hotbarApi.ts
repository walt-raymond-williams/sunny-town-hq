import type { HotbarSlot, StudentHotbar } from '../types/inventory'
import { authJson, jsonOptions } from './http'
import { normalizeInventoryItem, type InventoryItemResponse } from './inventoryApi'

interface HotbarSlotResponse {
  slot?: number
  item?: InventoryItemResponse | null
}

interface StudentHotbarResponse {
  slots?: HotbarSlotResponse[]
}

export async function getStudentHotbar(): Promise<StudentHotbar> {
  return normalizeHotbar(await authJson<StudentHotbarResponse>('/api/student/hotbar'))
}

export async function setStudentHotbarSlot(slot: number, itemKey: string): Promise<StudentHotbar> {
  return normalizeHotbar(await authJson<StudentHotbarResponse>(
    '/api/student/hotbar',
    jsonOptions('PUT', { slot, itemKey }),
  ))
}

export function normalizeHotbar(response?: StudentHotbarResponse): StudentHotbar {
  const bySlot = new Map<number, HotbarSlot>(
    (response?.slots || [])
      .map((slot): HotbarSlot => ({
        slot: slot.slot || 0,
        item: slot.item ? normalizeInventoryItem(slot.item) : null,
      }))
      .filter((slot) => slot.slot >= 1 && slot.slot <= 5)
      .map((slot) => [slot.slot, slot]),
  )

  return {
    slots: Array.from({ length: 5 }, (_, index) => bySlot.get(index + 1) || {
      slot: index + 1,
      item: null,
    }),
  }
}
