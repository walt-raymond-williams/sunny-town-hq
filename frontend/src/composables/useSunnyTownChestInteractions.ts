import { computed, ref } from 'vue'
import type { ContainerInventorySlots } from '../types/inventory'
import type { SunnyTownPlayer, SunnyTownWorldObject } from '../types/sunnyTown'

interface SunnyTownChestInteractionOptions {
  sendOpen?: (chest: SunnyTownWorldObject) => boolean
}

export function useSunnyTownChestInteractions(options: SunnyTownChestInteractionOptions = {}) {
  const nearbyChest = ref<SunnyTownWorldObject | null>(null)
  const activeChest = ref<SunnyTownWorldObject | null>(null)
  const chestOpen = ref(false)
  const chestError = ref('')
  const containerSlots = ref<ContainerInventorySlots | null>(null)
  const isLoadingChest = ref(false)
  const isTransferringChestSlot = ref(false)

  const canDepositIntoActiveChest = computed(() => activeChest.value?.storageRole === 'input' || activeChest.value?.storageRole === 'general')
  const canWithdrawFromActiveChest = computed(() => activeChest.value?.storageRole === 'general')

  function hasActiveOverlay(): boolean {
    return Boolean(activeChest.value)
  }

  function closeChest() {
    activeChest.value = null
    chestOpen.value = false
    chestError.value = ''
    containerSlots.value = null
    isLoadingChest.value = false
    isTransferringChestSlot.value = false
  }

  function inspectChest(chest: SunnyTownWorldObject | null): boolean {
    if (activeChest.value) {
      return true
    }
    if (!chest) {
      return false
    }
    activeChest.value = chest
    chestOpen.value = true
    chestError.value = ''
    containerSlots.value = null
    isLoadingChest.value = true

    if (!options.sendOpen?.(chest)) {
      isLoadingChest.value = false
      chestError.value = 'Sunny Town connection is not ready.'
    }
    return true
  }

  function applyContainerOpened(container: ContainerInventorySlots | null | undefined) {
    if (!activeChest.value) {
      return
    }
    if (!container) {
      chestError.value = 'Storage could not be loaded.'
      isLoadingChest.value = false
      return
    }
    containerSlots.value = container
    chestError.value = ''
    isLoadingChest.value = false
  }

  function startContainerTransfer() {
    isTransferringChestSlot.value = true
    chestError.value = ''
  }

  function applyContainerTransferCommitted(container: ContainerInventorySlots | null | undefined) {
    isTransferringChestSlot.value = false
    if (container) {
      containerSlots.value = container
    }
    chestError.value = ''
  }

  function applyContainerError(message: string) {
    if (!activeChest.value) {
      return
    }
    chestError.value = message
    isLoadingChest.value = false
    isTransferringChestSlot.value = false
  }

  function refreshNearby(nextNearbyChest: SunnyTownWorldObject | null) {
    nearbyChest.value = nextNearbyChest
    if (activeChest.value && !sameWorldObject(activeChest.value, nextNearbyChest)) {
      closeChest()
    }
  }

  return {
    activeChest,
    applyContainerError,
    applyContainerOpened,
    applyContainerTransferCommitted,
    canDepositIntoActiveChest,
    canWithdrawFromActiveChest,
    chestError,
    chestOpen,
    closeChest,
    containerSlots,
    hasActiveOverlay,
    inspectChest,
    isLoadingChest,
    isTransferringChestSlot,
    nearbyChest,
    refreshNearby,
    startContainerTransfer,
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
    if (!isInspectableChest(object)) {
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

function isInspectableChest(object: SunnyTownWorldObject): boolean {
  return Boolean(
    object.active &&
    object.kind === 'chest' &&
    object.source === 'fixture' &&
    (object.storageRole === 'output' || object.storageRole === 'input' || object.storageRole === 'general'),
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
