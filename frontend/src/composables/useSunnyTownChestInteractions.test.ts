import { describe, expect, it, vi } from 'vitest'
import type { SunnyTownPlayer, SunnyTownWorldObject } from '../types/sunnyTown'
import {
  nearestSunnyTownChest,
  useSunnyTownChestInteractions,
} from './useSunnyTownChestInteractions'

describe('useSunnyTownChestInteractions', () => {
  it('finds the nearest inspectable chest within its interaction radius', () => {
    const self = createPlayer({ x: 116, y: 116 })
    const far = createChest({ id: 'far', x: 260, y: 100 })
    const near = createChest({ id: 'near', x: 96, y: 96, interactionRadius: 48, storageRole: 'input' })
    const inactive = createChest({ id: 'inactive', x: 112, y: 112, active: false })
    const rock = { ...createChest({ id: 'rock', x: 112, y: 112 }), kind: 'rock_node' as const }

    expect(nearestSunnyTownChest([far, inactive, rock, near], self)?.id).toBe('near')
    expect(nearestSunnyTownChest([far], self)).toBeNull()
    expect(nearestSunnyTownChest([near], null)).toBeNull()
  })

  it('opens a chest and requests its container slots', () => {
    const sendOpen = vi.fn().mockReturnValue(true)
    const interactions = useSunnyTownChestInteractions({ sendOpen })
    const chest = createChest()

    expect(interactions.inspectChest(chest)).toBe(true)

    expect(interactions.activeChest.value?.id).toBe('cookie-shop-output-chest')
    expect(sendOpen).toHaveBeenCalledWith(chest)
    expect(interactions.isLoadingChest.value).toBe(true)

    interactions.applyContainerOpened({
      containerId: 'fixture:sunny-town-main:sunny-town-house-1:cookie-shop-output-chest',
      slotCount: 1,
      revision: 1,
      slots: [{ slotIndex: 0, item: null }],
    })

    expect(interactions.containerSlots.value?.slotCount).toBe(1)
    expect(interactions.isLoadingChest.value).toBe(false)
  })

  it('closes an active chest when nearby chest changes', () => {
    const interactions = useSunnyTownChestInteractions({
      sendOpen: vi.fn().mockReturnValue(true),
    })

    interactions.inspectChest(createChest({ id: 'first' }))
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
