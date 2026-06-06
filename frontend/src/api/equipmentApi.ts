import { authJson, jsonOptions } from './http'
import type { EquipmentItem, EquipmentSlot, StudentEquipment } from '../types/inventory'

interface EquipmentItemResponse {
  key?: string
  name?: string
  description?: string
  equipSlot?: EquipmentSlot
  visualKey?: string
}

interface EquipmentSlotResponse {
  slot?: EquipmentSlot
  item?: EquipmentItemResponse | null
}

interface StudentEquipmentResponse {
  slots?: EquipmentSlotResponse[]
}

export async function getStudentEquipment(): Promise<StudentEquipment> {
  const response = await authJson<StudentEquipmentResponse>('/api/student/equipment')
  return normalizeStudentEquipment(response)
}

export async function equipStudentItem(slot: EquipmentSlot, itemKey: string): Promise<StudentEquipment> {
  const response = await authJson<StudentEquipmentResponse>(
    '/api/student/equipment/equip',
    jsonOptions('POST', { slot, itemKey }),
  )
  return normalizeStudentEquipment(response)
}

export async function unequipStudentItem(slot: EquipmentSlot): Promise<StudentEquipment> {
  const response = await authJson<StudentEquipmentResponse>(
    '/api/student/equipment/unequip',
    jsonOptions('POST', { slot }),
  )
  return normalizeStudentEquipment(response)
}

function normalizeStudentEquipment(response: StudentEquipmentResponse): StudentEquipment {
  return {
    slots: (response.slots || [])
      .filter((slot): slot is EquipmentSlotResponse & { slot: EquipmentSlot } => slot.slot === 'gear' || slot.slot === 'accessory' || slot.slot === 'tool')
      .map((slot) => ({
        slot: slot.slot,
        item: slot.item ? normalizeEquipmentItem(slot.item) : null,
      })),
  }
}

function normalizeEquipmentItem(item: EquipmentItemResponse): EquipmentItem {
  return {
    key: item.key || '',
    name: item.name || '',
    description: item.description || '',
    equipSlot: item.equipSlot || 'gear',
    visualKey: item.visualKey || item.key || '',
  }
}
