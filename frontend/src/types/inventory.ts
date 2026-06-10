export interface InventoryItem {
  key: string
  name: string
  description: string
  quantity: number
  equipSlot: EquipmentSlot | ''
  visualKey: string
  iconKey: string
  maxStack: number
  category: string
  equipped: boolean
}

export interface StudentInventory {
  items: InventoryItem[]
}

export interface InventorySlot {
  slotIndex: number
  item: InventoryItem | null
}

export interface StudentInventorySlots {
  slotCount: number
  slots: InventorySlot[]
  items: InventoryItem[]
}

export interface ContainerInventorySlots {
  containerId: string
  slotCount: number
  revision: number
  slots: InventorySlot[]
}

export type InventoryStorageKind = 'player_inventory' | 'container'

export interface InventoryStorageRef {
  kind: InventoryStorageKind
  containerId?: string
  slotIndex: number
}

export type InventoryMoveMode = 'move' | 'swap' | 'merge' | 'auto'

export interface InventoryMoveRequest {
  source: InventoryStorageRef
  destination: InventoryStorageRef
  mode: InventoryMoveMode
}

export interface InventorySlotItem {
  key: string
  name: string
  description: string
  iconKey: string
  quantity?: number
  maxStack?: number
  category?: string
}

export interface HotbarSlot {
  slot: number
  item: InventoryItem | null
}

export interface StudentHotbar {
  slots: HotbarSlot[]
}

export interface CraftingIngredient {
  itemKey: string
  name: string
  description: string
  iconKey: string
  maxStack: number
  category: string
  required: number
  owned: number
}

export interface CraftingRecipe {
  key: string
  name: string
  description: string
  outputKey: string
  outputName: string
  outputIconKey: string
  outputMaxStack: number
  outputCategory: string
  quantity: number
  canCraft: boolean
  ingredients: CraftingIngredient[]
}

export interface CraftRecipeResult {
  inventory: StudentInventorySlots
  recipes: CraftingRecipe[]
}

export type EquipmentSlot = 'gear' | 'accessory' | 'tool'

export interface EquipmentItem {
  key: string
  name: string
  description: string
  equipSlot: EquipmentSlot
  visualKey: string
  iconKey: string
  maxStack: number
  category: string
}

export interface EquippedSlot {
  slot: EquipmentSlot
  item: EquipmentItem | null
}

export interface StudentEquipment {
  slots: EquippedSlot[]
}
