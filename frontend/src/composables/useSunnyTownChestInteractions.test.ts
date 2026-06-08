import { describe, expect, it, vi } from 'vitest'
import type { SunnyTownPlayer, SunnyTownWorldObject } from '../types/sunnyTown'
import {
  nearestSunnyTownChest,
  useSunnyTownChestInteractions,
} from './useSunnyTownChestInteractions'

describe('useSunnyTownChestInteractions', () => {
  it('finds the nearest inspectable output chest within its interaction radius', () => {
    const self = createPlayer({ x: 116, y: 116 })
    const far = createChest({ id: 'far', x: 260, y: 100 })
    const near = createChest({ id: 'near', x: 96, y: 96, interactionRadius: 48 })
    const inactive = createChest({ id: 'inactive', x: 112, y: 112, active: false })
    const input = createChest({ id: 'input', x: 112, y: 112, storageRole: 'input' })

    expect(nearestSunnyTownChest([far, inactive, input, near], self)?.id).toBe('near')
    expect(nearestSunnyTownChest([far], self)).toBeNull()
    expect(nearestSunnyTownChest([near], null)).toBeNull()
  })

  it('opens a chest and loads its backing shop stock', async () => {
    const loadShopStock = vi.fn().mockResolvedValue({
      shopId: 'cookie-keeper-shop',
      items: [{ itemKey: 'cookie', quantity: 12, capacity: 64 }],
    })
    const interactions = useSunnyTownChestInteractions({ loadShopStock })
    const chest = createChest()

    await expect(interactions.inspectChest(chest)).resolves.toBe(true)

    expect(interactions.activeChest.value?.id).toBe('cookie-shop-output-chest')
    expect(loadShopStock).toHaveBeenCalledWith('cookie-keeper-shop')
    expect(interactions.activeChestItemKey.value).toBe('cookie')
    expect(interactions.activeChestQuantity.value).toBe(12)
    expect(interactions.activeChestCapacity.value).toBe(64)
    expect(interactions.isLoadingChest.value).toBe(false)
  })

  it('closes an active chest when nearby chest changes', async () => {
    const interactions = useSunnyTownChestInteractions({
      loadShopStock: vi.fn().mockResolvedValue({ shopId: 'shop-1', items: [] }),
    })

    await interactions.inspectChest(createChest({ id: 'first' }))
    interactions.refreshNearby(createChest({ id: 'second' }))

    expect(interactions.activeChest.value).toBeNull()
  })
})

function createPlayer(overrides: Partial<SunnyTownPlayer> = {}): SunnyTownPlayer {
  return {
    id: 'self',
    displayName: 'Self',
    x: 0,
    y: 0,
    facing: 'down',
    moving: false,
    avatarId: 'default',
    lastProcessedSeq: 0,
    ...overrides,
  }
}

function createChest(overrides: Partial<SunnyTownWorldObject> = {}): SunnyTownWorldObject {
  return {
    id: 'cookie-shop-output-chest',
    kind: 'chest',
    source: 'fixture',
    name: 'Cookie Shop Output Chest',
    itemKey: 'cookie',
    shopId: 'cookie-keeper-shop',
    storageRole: 'output',
    x: 96,
    y: 96,
    width: 32,
    height: 32,
    interactionRadius: 56,
    active: true,
    collision: true,
    breakable: false,
    reservesPlacement: true,
    ...overrides,
  }
}
