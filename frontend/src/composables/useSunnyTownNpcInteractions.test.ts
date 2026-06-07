import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import type { Assignment } from '../types/assignment'
import type { InventoryItem } from '../types/inventory'
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

  it('loads schoolwork and submits answers through injected APIs', async () => {
    const nextAssignment = createAssignment({ id: 1, prompt: 'First' })
    const followupAssignment = createAssignment({ id: 2, prompt: 'Second' })
    const loadNextAssignment = vi.fn()
      .mockResolvedValueOnce(nextAssignment)
      .mockResolvedValueOnce(followupAssignment)
    const submitStudentAnswer = vi.fn().mockResolvedValue(createAssignment())
    const interactions = useSunnyTownNpcInteractions({
      loadNextAssignment,
      submitStudentAnswer,
    })

    interactions.openSchoolworkMenu(createNpc({ activity: { type: 'schoolwork' } }))
    await interactions.startSchoolwork()
    interactions.schoolworkAnswer.value = '  done  '
    await interactions.submitSchoolworkAnswer()

    expect(interactions.schoolworkOpen.value).toBe(true)
    expect(submitStudentAnswer).toHaveBeenCalledWith(1, 'done')
    expect(interactions.schoolworkNotice.value).toBe('Answer submitted.')
    expect(interactions.schoolworkAnswer.value).toBe('')
    expect(interactions.schoolworkAssignment.value?.id).toBe(2)
  })

  it('opens trade and applies purchases through injected inventory hooks', async () => {
    const starBalance = ref(5)
    const loadInventory = vi.fn()
    const setInventoryItems = vi.fn()
    const purchaseShopItem = vi.fn().mockResolvedValue({
      starBalance: 2,
      inventory: {
        items: [createInventoryItem({ key: 'star_cap', quantity: 1 })],
      },
    })
    const interactions = useSunnyTownNpcInteractions({
      loadInventory,
      purchaseShopItem,
      setInventoryItems,
      starBalance,
    })

    interactions.openShopMenu(createNpc({ shop: { id: 'shop-1', items: [] } }))
    await interactions.openTrade()
    await interactions.buyShopItem('star_cap')

    expect(interactions.shopOpen.value).toBe(true)
    expect(loadInventory).toHaveBeenCalledOnce()
    expect(purchaseShopItem).toHaveBeenCalledWith({
      shopId: 'shop-1',
      itemKey: 'star_cap',
      quantity: 1,
    })
    expect(starBalance.value).toBe(2)
    expect(setInventoryItems).toHaveBeenCalledWith([createInventoryItem({ key: 'star_cap', quantity: 1 })])
    expect(interactions.shopNotice.value).toBe('Purchased.')
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

function createAssignment(overrides: Partial<Assignment> = {}): Assignment {
  return {
    id: 1,
    category: 'MATH',
    prompt: 'What is 1 + 1?',
    expected_answer: '2',
    created_at: '2026-06-07T00:00:00Z',
    current_attempt: null,
    attempts: [],
    ...overrides,
  }
}

function createInventoryItem(overrides: Partial<InventoryItem> = {}): InventoryItem {
  return {
    key: 'rock',
    name: 'Rock',
    description: 'A rock.',
    quantity: 1,
    equipSlot: '',
    visualKey: '',
    equipped: false,
    ...overrides,
  }
}
