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
  inventory: StudentInventory
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
