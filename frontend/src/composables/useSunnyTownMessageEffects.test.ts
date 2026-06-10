import { ref } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useSunnyTownMessageEffects } from './useSunnyTownMessageEffects'

function createEffects() {
  const inventoryStore = {
    loadCraftingRecipes: vi.fn(),
    setItemQuantity: vi.fn(),
  }
  const refreshProgression = vi.fn()
  const refs = {
    craftingPanelOpen: ref(false),
    error: ref(''),
    gameToast: ref(''),
    progressionPanelOpen: ref(false),
    starBalance: ref(10),
  }
  return {
    effects: useSunnyTownMessageEffects({
      ...refs,
      inventoryStore,
      refreshProgression,
      setTimeoutFn: globalThis.setTimeout,
    }),
    inventoryStore,
    refreshProgression,
    refs,
  }
}

describe('useSunnyTownMessageEffects', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('applies reward commits and clears the toast later', () => {
    vi.useFakeTimers()
    const { effects, refs } = createEffects()

    effects.applyRewardCommitted({
      type: 'reward_committed',
      amount: 2,
      newStarBalance: 12,
    })

    expect(refs.starBalance.value).toBe(12)
    expect(refs.gameToast.value).toBe('+2 star')

    vi.advanceTimersByTime(1200)

    expect(refs.gameToast.value).toBe('')
  })

  it('falls back to adding reward amount to current star balance', () => {
    const { effects, refs } = createEffects()

    effects.applyRewardCommitted({
      type: 'reward_committed',
      amount: 3,
    })

    expect(refs.starBalance.value).toBe(13)
  })

  it('applies resource commits and refreshes crafting when the panel is open', () => {
    const { effects, inventoryStore, refs } = createEffects()
    refs.craftingPanelOpen.value = true

    effects.applyResourceCommitted({
      type: 'resource_committed',
      amount: 2,
      quantity: 5,
      resourceKey: 'rock',
    })

    expect(refs.gameToast.value).toBe('+2 rock')
    expect(inventoryStore.setItemQuantity).toHaveBeenCalledWith('rock', 5)
    expect(inventoryStore.loadCraftingRecipes).toHaveBeenCalledOnce()
  })

  it('refreshes progression after resource commits only when the character panel is open', () => {
    const { effects, refreshProgression, refs } = createEffects()

    effects.applyResourceCommitted({
      type: 'resource_committed',
      amount: 1,
      quantity: 3,
      resourceKey: 'rock',
    })

    expect(refreshProgression).not.toHaveBeenCalled()

    refs.progressionPanelOpen.value = true
    effects.applyResourceCommitted({
      type: 'resource_committed',
      amount: 1,
      quantity: 4,
      resourceKey: 'rock',
    })

    expect(refreshProgression).toHaveBeenCalledOnce()
  })

  it('applies placed and removed object inventory effects', () => {
    const { effects, inventoryStore, refs } = createEffects()
    refs.craftingPanelOpen.value = true

    effects.applyPlacedObjectCommitted({
      type: 'map_object_placed',
      quantity: 1,
      resourceKey: 'stone_block',
    })
    effects.applyRemovedObjectCommitted({
      type: 'map_object_removed',
      quantity: 2,
      resourceKey: 'stone_block',
    })

    expect(inventoryStore.setItemQuantity).toHaveBeenNthCalledWith(1, 'stone_block', 1)
    expect(inventoryStore.setItemQuantity).toHaveBeenNthCalledWith(2, 'stone_block', 2)
    expect(refs.gameToast.value).toBe('+1 stone_block')
    expect(inventoryStore.loadCraftingRecipes).toHaveBeenCalledOnce()
  })

  it('sets user-facing errors for failed commits', () => {
    const { effects, refs } = createEffects()

    effects.applyRewardFailed()
    expect(refs.error.value).toBe('That star could not be saved. Try again in a moment.')

    effects.applyResourceFailed()
    expect(refs.error.value).toBe('That resource could not be saved. Try again in a moment.')
  })
})
