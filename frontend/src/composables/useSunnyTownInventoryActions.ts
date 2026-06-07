import { computed, ref } from 'vue'
import type {
  EquipmentSlot,
  HotbarSlot,
  InventoryItem,
} from '../types/inventory'

interface SunnyTownInventoryStore {
  hotbarSlots: HotbarSlot[]
  items: InventoryItem[]
  craftRecipe: (recipeKey: string) => Promise<void>
  equipItem: (slot: EquipmentSlot, itemKey: string) => Promise<void>
  loadCraftingRecipes: () => Promise<void>
  loadHotbar: () => Promise<void>
  loadInventory: () => Promise<void>
  setHotbarSlot: (slot: number, itemKey: string) => Promise<void>
  unequipItem: (slot: EquipmentSlot) => Promise<void>
}

interface SunnyTownInventoryActionOptions {
  onEquipmentChanged: () => void
  onHotbarSelectionChanged: () => void
  onHotbarUpdated: () => void
}

export function hotbarIndexForEvent(event: Pick<KeyboardEvent, 'code'>): number | null {
  if (!/^Digit[1-5]$/.test(event.code)) {
    return null
  }
  return Number(event.code.slice(5)) - 1
}

export function useSunnyTownInventoryActions(
  inventoryStore: SunnyTownInventoryStore,
  options: SunnyTownInventoryActionOptions,
) {
  const inventoryOpen = ref(false)
  const craftingPanelOpen = ref(false)
  const showAllCraftingRecipes = ref(false)
  const selectedHotbarIndex = ref(0)

  const stoneBlockQuantity = computed(() => inventoryStore.items.find((item) => item.key === 'stone_block')?.quantity || 0)
  const selectedHotbarSlot = computed(() => inventoryStore.hotbarSlots[selectedHotbarIndex.value] || null)
  const selectedHotbarItem = computed(() => selectedHotbarSlot.value?.item || null)
  const selectedHotbarItemKey = computed(() => selectedHotbarItem.value?.key || '')
  const placingStoneBlock = computed(() => selectedHotbarItemKey.value === 'stone_block' && stoneBlockQuantity.value > 0)

  async function toggleInventory() {
    inventoryOpen.value = !inventoryOpen.value
    if (inventoryOpen.value) {
      craftingPanelOpen.value = true
      await inventoryStore.loadInventory()
      await inventoryStore.loadHotbar()
      await inventoryStore.loadCraftingRecipes()
    }
  }

  async function toggleCraftingPanel() {
    craftingPanelOpen.value = !craftingPanelOpen.value
    if (craftingPanelOpen.value) {
      await inventoryStore.loadCraftingRecipes()
    }
  }

  async function craftInventoryRecipe(recipeKey: string) {
    await inventoryStore.craftRecipe(recipeKey)
    await inventoryStore.loadHotbar()
  }

  function selectHotbarSlot(index: number) {
    selectedHotbarIndex.value = index
    options.onHotbarSelectionChanged()
  }

  async function equipInventoryItem(itemKey: string, slot: EquipmentSlot | '') {
    if (!slot) {
      return
    }
    await inventoryStore.equipItem(slot, itemKey)
    options.onEquipmentChanged()
  }

  async function unequipInventorySlot(slot: EquipmentSlot) {
    await inventoryStore.unequipItem(slot)
    options.onEquipmentChanged()
  }

  async function assignInventoryItemToSelectedHotbarSlot(itemKey: string) {
    await inventoryStore.setHotbarSlot(selectedHotbarIndex.value + 1, itemKey)
    options.onHotbarUpdated()
  }

  async function clearSelectedHotbarSlot() {
    await inventoryStore.setHotbarSlot(selectedHotbarIndex.value + 1, '')
    options.onHotbarUpdated()
  }

  return {
    assignInventoryItemToSelectedHotbarSlot,
    clearSelectedHotbarSlot,
    craftInventoryRecipe,
    craftingPanelOpen,
    equipInventoryItem,
    inventoryOpen,
    placingStoneBlock,
    selectedHotbarIndex,
    selectedHotbarItemKey,
    selectHotbarSlot,
    showAllCraftingRecipes,
    stoneBlockQuantity,
    toggleCraftingPanel,
    toggleInventory,
    unequipInventorySlot,
  }
}
