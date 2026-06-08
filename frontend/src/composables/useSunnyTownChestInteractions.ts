import { computed, ref } from 'vue'
import type { SunnyTownPlayer, SunnyTownWorldObject } from '../types/sunnyTown'

interface ShopStockResult {
  shopId: string
  items: Array<{
    itemKey: string
    quantity: number
    capacity: number
  }>
}

interface SunnyTownChestInteractionOptions {
  loadShopStock?: (shopId: string) => Promise<ShopStockResult>
}

export function useSunnyTownChestInteractions(options: SunnyTownChestInteractionOptions = {}) {
  const nearbyChest = ref<SunnyTownWorldObject | null>(null)
  const activeChest = ref<SunnyTownWorldObject | null>(null)
  const chestOpen = ref(false)
  const chestError = ref('')
  const chestStock = ref<Record<string, number>>({})
  const chestStockCapacity = ref<Record<string, number>>({})
  const isLoadingChest = ref(false)

  const activeChestItemKey = computed(() => activeChest.value?.itemKey || '')
  const activeChestQuantity = computed(() => (
    activeChestItemKey.value ? chestStock.value[activeChestItemKey.value] ?? 0 : 0
  ))
  const activeChestCapacity = computed(() => (
    activeChestItemKey.value ? chestStockCapacity.value[activeChestItemKey.value] ?? 0 : 0
  ))

  function hasActiveOverlay(): boolean {
    return Boolean(activeChest.value)
  }

  function closeChest() {
    activeChest.value = null
    chestOpen.value = false
    chestError.value = ''
    chestStock.value = {}
    chestStockCapacity.value = {}
    isLoadingChest.value = false
  }

  async function inspectChest(chest: SunnyTownWorldObject | null): Promise<boolean> {
    if (activeChest.value) {
      return true
    }
    if (!chest || !chest.shopId) {
      return false
    }
    activeChest.value = chest
    chestOpen.value = true
    chestError.value = ''
    chestStock.value = {}
    chestStockCapacity.value = {}
    isLoadingChest.value = true

    try {
      const stock = await loadShopStock(chest.shopId)
      chestStock.value = Object.fromEntries(stock.items.map((item) => [item.itemKey, item.quantity]))
      chestStockCapacity.value = Object.fromEntries(stock.items.map((item) => [item.itemKey, item.capacity]))
    } catch (caught) {
      chestError.value = errorMessage(caught)
    } finally {
      isLoadingChest.value = false
    }
    return true
  }

  function refreshNearby(nextNearbyChest: SunnyTownWorldObject | null) {
    nearbyChest.value = nextNearbyChest
    if (activeChest.value && !sameWorldObject(activeChest.value, nextNearbyChest)) {
      closeChest()
    }
  }

  async function loadShopStock(shopId: string): Promise<ShopStockResult> {
    if (options.loadShopStock) {
      return options.loadShopStock(shopId)
    }
    const shopApi = await import('../api/shopApi')
    return shopApi.getShopStock(shopId)
  }

  return {
    activeChest,
    activeChestCapacity,
    activeChestItemKey,
    activeChestQuantity,
    chestError,
    chestOpen,
    chestStock,
    chestStockCapacity,
    closeChest,
    hasActiveOverlay,
    inspectChest,
    isLoadingChest,
    nearbyChest,
    refreshNearby,
  }
}

export function nearestSunnyTownChest(
  objects: SunnyTownWorldObject[],
  self: SunnyTownPlayer | null | undefined,
): SunnyTownWorldObject | null {
  if (!self) {
    return null
  }

  let nearest: SunnyTownWorldObject | null = null
  let nearestDistance = Number.POSITIVE_INFINITY
  for (const object of objects) {
    if (!isInspectableOutputChest(object)) {
      continue
    }
    const center = worldObjectCenter(object)
    const distance = Math.hypot(self.x - center.x, self.y - center.y)
    const radius = object.interactionRadius || 56
    if (distance <= radius && distance < nearestDistance) {
      nearest = object
      nearestDistance = distance
    }
  }
  return nearest
}

function isInspectableOutputChest(object: SunnyTownWorldObject): boolean {
  return Boolean(
    object.active &&
    object.kind === 'chest' &&
    object.source === 'fixture' &&
    object.storageRole === 'output' &&
    object.shopId &&
    object.itemKey,
  )
}

function worldObjectCenter(object: SunnyTownWorldObject): { x: number; y: number } {
  if (object.radius && object.radius > 0) {
    return { x: object.x, y: object.y }
  }
  return {
    x: object.x + (object.width || 0) / 2,
    y: object.y + (object.height || 0) / 2,
  }
}

function sameWorldObject(first: SunnyTownWorldObject, second: SunnyTownWorldObject | null): boolean {
  return Boolean(second && first.source === second.source && first.id === second.id)
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
