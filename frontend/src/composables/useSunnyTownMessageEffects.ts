import type { Ref } from 'vue'
import type { SunnyTownServerMessage } from '../types/sunnyTown'

interface SunnyTownMessageInventoryStore {
  loadCraftingRecipes: () => Promise<void>
  setItemQuantity: (itemKey: string, quantity: number) => void
}

interface SunnyTownMessageEffectsOptions {
  craftingPanelOpen: Ref<boolean>
  error: Ref<string>
  gameToast: Ref<string>
  inventoryStore: SunnyTownMessageInventoryStore
  starBalance: Ref<number>
  setTimeoutFn?: typeof window.setTimeout
}

const toastDurationMs = 1200

export function useSunnyTownMessageEffects(options: SunnyTownMessageEffectsOptions) {
  const setTimeoutFn = options.setTimeoutFn || window.setTimeout

  function showToast(message: string) {
    options.gameToast.value = message
    setTimeoutFn(() => {
      options.gameToast.value = ''
    }, toastDurationMs)
  }

  function applyRewardCommitted(message: SunnyTownServerMessage) {
    options.starBalance.value = message.newStarBalance ?? options.starBalance.value + (message.amount || 0)
    showToast(`+${message.amount || 1} star`)
  }

  function applyRewardFailed() {
    options.error.value = 'That star could not be saved. Try again in a moment.'
  }

  function applyResourceCommitted(message: SunnyTownServerMessage) {
    const amount = message.amount || 1
    const resourceKey = message.resourceKey || 'rock'
    showToast(`+${amount} ${resourceKey}`)
    if (message.quantity !== undefined) {
      options.inventoryStore.setItemQuantity(resourceKey, message.quantity)
    }
    refreshCraftingIfOpen()
  }

  function applyResourceFailed() {
    options.error.value = 'That resource could not be saved. Try again in a moment.'
  }

  function applyPlacedObjectCommitted(message: SunnyTownServerMessage) {
    if (!message.resourceKey || message.quantity === undefined) {
      return
    }
    options.inventoryStore.setItemQuantity(message.resourceKey, message.quantity)
    showToast('Stone block placed')
  }

  function applyRemovedObjectCommitted(message: SunnyTownServerMessage) {
    if (message.resourceKey && message.quantity !== undefined) {
      options.inventoryStore.setItemQuantity(message.resourceKey, message.quantity)
      showToast('+1 stone_block')
    }
    refreshCraftingIfOpen()
  }

  function refreshCraftingIfOpen() {
    if (options.craftingPanelOpen.value) {
      void options.inventoryStore.loadCraftingRecipes()
    }
  }

  return {
    applyPlacedObjectCommitted,
    applyRemovedObjectCommitted,
    applyResourceCommitted,
    applyResourceFailed,
    applyRewardCommitted,
    applyRewardFailed,
  }
}
