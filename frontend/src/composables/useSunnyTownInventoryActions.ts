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
  dropInventorySlotOnEquipment: (slot: EquipmentSlot, equip: (slot: EquipmentSlot, itemKey: string) => Promise<void>) => Promise<boolean>
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

export type SunnyTownInventoryMenuTab = 'inventory' | 'crafting'

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
  const inventoryMenuTab = ref<SunnyTownInventoryMenuTab>('inventory')
  const craftingPanelOpen = ref(false)
  const showAllCraftingRecipes = ref(true)
  const selectedHotbarIndex = ref(0)

  const stoneBlockQuantity = computed(() => inventoryStore.items.find((item) => item.key === 'stone_block')?.quantity || 0)
  const selectedHotbarSlot = computed(() => inventoryStore.hotbarSlots[selectedHotbarIndex.value] || null)
  const selectedHotbarItem = computed(() => selectedHotbarSlot.value?.item || null)
  const selectedHotbarItemKey = computed(() => selectedHotbarItem.value?.key || '')
  const placingStoneBlock = computed(() => selectedHotbarItemKey.value === 'stone_block' && stoneBlockQuantity.value > 0)

  async function toggleInventory() {
    inventoryOpen.value = !inventoryOpen.value
    if (inventoryOpen.value) {
      inventoryMenuTab.value = 'inventory'
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

  async function setInventoryMenuTab(tab: SunnyTownInventoryMenuTab) {
    inventoryMenuTab.value = tab
    if (tab === 'crafting') {
      craftingPanelOpen.value = true
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

  async function equipInventorySlotDrop(slot: EquipmentSlot) {
    await inventoryStore.dropInventorySlotOnEquipment(slot, async (equipmentSlot, itemKey) => {
      await inventoryStore.equipItem(equipmentSlot, itemKey)
      options.onEquipmentChanged()
    })
  }

  async function unequipInventorySlot(slot: EquipmentSlot) {
    await inventoryStore.unequipItem(slot)
    options.onEquipmentChanged()
  }

  async function clearSelectedHotbarSlot() {
    await inventoryStore.setHotbarSlot(selectedHotbarIndex.value + 1, '')
    options.onHotbarUpdated()
  }

  return {
    clearSelectedHotbarSlot,
    craftInventoryRecipe,
    craftingPanelOpen,
    equipInventorySlotDrop,
    inventoryMenuTab,
    inventoryOpen,
    placingStoneBlock,
    selectedHotbarIndex,
    selectedHotbarItemKey,
    selectHotbarSlot,
    setInventoryMenuTab,
    showAllCraftingRecipes,
    stoneBlockQuantity,
    toggleCraftingPanel,
    toggleInventory,
    unequipInventorySlot,
  }
}
