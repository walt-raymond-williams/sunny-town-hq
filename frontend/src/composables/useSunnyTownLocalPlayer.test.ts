import { describe, expect, it } from 'vitest'
import type { SunnyTownMap, SunnyTownPlayer } from '../types/sunnyTown'
import { useSunnyTownLocalPlayer } from './useSunnyTownLocalPlayer'

describe('useSunnyTownLocalPlayer', () => {
  it('uses the snapshot as current self until prediction state exists', () => {
    const localPlayer = useSunnyTownLocalPlayer()
    const self = player({ id: 'self', x: 20 })

    expect(localPlayer.current([self], 'self')).toEqual(self)
    expect(localPlayer.local()).toBeNull()
  })

  it('syncs display fields from snapshots without overwriting predicted position', () => {
    const localPlayer = useSunnyTownLocalPlayer()
    const first = player({ id: 'self', x: 20, displayName: 'First', lastProcessedSeq: 1 })
    const next = player({
      id: 'self',
      x: 200,
      displayName: 'Next',
      avatarId: 'updated',
      equipment: { gear: 'sunny_hoodie' },
      lastProcessedSeq: 2,
    })

    localPlayer.syncFromSnapshot(first)
    localPlayer.predict(first, { up: false, down: false, left: false, right: true }, map(), [], 1)
    localPlayer.syncFromSnapshot(next)

    expect(localPlayer.local()).toEqual({
      ...first,
      x: 170,
      y: 14,
      displayName: 'Next',
      avatarId: 'updated',
      equipment: { gear: 'sunny_hoodie' },
      lastProcessedSeq: 2,
      facing: 'right',
      moving: true,
    })
  })

  it('renders remote players plus a copied rendered self', () => {
    const localPlayer = useSunnyTownLocalPlayer()
    const self = player({ id: 'self', x: 20 })
    const remote = player({ id: 'remote', x: 50 })

    const rendered = localPlayer.renderedPlayers(self, [remote])

    expect(rendered).toHaveLength(2)
    expect(rendered[0]).toEqual(remote)
    expect(rendered[1]).toEqual(self)
    expect(rendered[1]).not.toBe(self)
  })

  it('applies equipment visuals to local and rendered self state', () => {
    const localPlayer = useSunnyTownLocalPlayer()
    const self = player({ id: 'self' })

    localPlayer.renderedPlayers(self, [])
    localPlayer.applyEquipment({ tool: 'pickaxe' })

    expect(localPlayer.local()?.equipment).toEqual({ tool: 'pickaxe' })
    expect(localPlayer.renderedPlayers(self, [])[0]?.equipment).toEqual({ tool: 'pickaxe' })
  })

  it('clears local and rendered state', () => {
    const localPlayer = useSunnyTownLocalPlayer()
    const self = player({ id: 'self' })

    localPlayer.syncFromSnapshot(self)
    localPlayer.clear()

    expect(localPlayer.local()).toBeNull()
    expect(localPlayer.renderedPlayers(null, [])).toEqual([])
  })
})

function player(overrides: Partial<SunnyTownPlayer>): SunnyTownPlayer {
  return {
    id: 'player',
    displayName: 'Player',
    x: 0,
    y: 0,
    facing: 'down',
    moving: false,
    avatarId: 'default',
    lastProcessedSeq: 0,
    ...overrides,
  }
}

function map(): SunnyTownMap {
  return {
    id: 'test-map',
    name: 'Test Map',
    tileSize: 32,
    width: 20,
    height: 20,
    spawns: [],
    blockedRects: [],
    starSpawns: [],
    portals: [],
    npcs: [],
    resourceNodes: [],
  }
}
