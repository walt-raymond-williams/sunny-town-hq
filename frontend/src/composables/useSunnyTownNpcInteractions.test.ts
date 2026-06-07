import { describe, expect, it } from 'vitest'
import type { SunnyTownNpc, SunnyTownPlayer } from '../types/sunnyTown'
import { nearestSunnyTownNpc, useSunnyTownNpcInteractions } from './useSunnyTownNpcInteractions'

describe('useSunnyTownNpcInteractions', () => {
  it('opens and advances dialogue before closing it', () => {
    const interactions = useSunnyTownNpcInteractions()
    const npc = createNpc({ dialogue: ['Hello', 'Bye'] })

    expect(interactions.interactWith(npc)).toEqual({ handled: true, openedSchoolwork: false })
    expect(interactions.activeDialogueLine.value).toBe('Hello')
    expect(interactions.dialogueProgress.value).toBe('1/2')

    interactions.interactWith(null)
    expect(interactions.activeDialogueLine.value).toBe('Bye')
    expect(interactions.dialogueProgress.value).toBe('2/2')

    interactions.interactWith(null)
    expect(interactions.activeDialogueNpc.value).toBeNull()
  })

  it('opens shop and schoolwork overlays based on npc capabilities', () => {
    const interactions = useSunnyTownNpcInteractions()
    const shopNpc = createNpc({ shop: { id: 'shop-1', items: [] } })
    const schoolNpc = createNpc({ activity: { type: 'schoolwork' } })

    expect(interactions.interactWith(shopNpc)).toEqual({ handled: true, openedSchoolwork: false })
    expect(interactions.activeShopNpc.value?.id).toBe(shopNpc.id)

    interactions.closeOverlays()
    expect(interactions.interactWith(schoolNpc)).toEqual({ handled: true, openedSchoolwork: true })
    expect(interactions.activeSchoolworkNpc.value?.id).toBe(schoolNpc.id)
  })

  it('resets overlay state when closing overlays', () => {
    const interactions = useSunnyTownNpcInteractions()
    const npc = createNpc({ shop: { id: 'shop-1', items: [] } })

    interactions.interactWith(npc)
    interactions.shopOpen.value = true
    interactions.shopError.value = 'Nope'
    interactions.isPurchasing.value = true
    interactions.closeOverlays()

    expect(interactions.activeShopNpc.value).toBeNull()
    expect(interactions.shopOpen.value).toBe(false)
    expect(interactions.shopError.value).toBe('')
    expect(interactions.isPurchasing.value).toBe(false)
  })

  it('closes active overlays when the nearby npc changes', () => {
    const interactions = useSunnyTownNpcInteractions()
    const first = createNpc({ id: 'first', shop: { id: 'shop-1', items: [] } })
    const second = createNpc({ id: 'second' })

    interactions.interactWith(first)
    interactions.refreshNearby(second)

    expect(interactions.nearbyNpc.value?.id).toBe('second')
    expect(interactions.activeShopNpc.value).toBeNull()
  })

  it('finds the nearest npc within range', () => {
    const self = createPlayer({ x: 100, y: 100 })
    const far = createNpc({ id: 'far', x: 200, y: 100 })
    const near = createNpc({ id: 'near', x: 110, y: 100 })

    expect(nearestSunnyTownNpc([far, near], self, 54)?.id).toBe('near')
    expect(nearestSunnyTownNpc([far], self, 54)).toBeNull()
    expect(nearestSunnyTownNpc([near], null, 54)).toBeNull()
  })
})

function createNpc(overrides: Partial<SunnyTownNpc> = {}): SunnyTownNpc {
  return {
    id: 'npc',
    name: 'NPC',
    x: 0,
    y: 0,
    facing: 'down',
    spriteKey: 'keeper',
    dialogue: ['Hello'],
    ...overrides,
  }
}

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
