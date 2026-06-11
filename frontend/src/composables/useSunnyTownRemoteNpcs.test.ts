import { describe, expect, it } from 'vitest'
import type { SunnyTownNpc } from '../types/sunnyTown'
import { interpolateRemoteNpc, useSunnyTownRemoteNpcs } from './useSunnyTownRemoteNpcs'

describe('useSunnyTownRemoteNpcs', () => {
  it('smooths NPCs at the delayed render time', () => {
    let now = 1_250
    const remoteNpcs = useSunnyTownRemoteNpcs({
      interpolationDelayMs: 150,
      now: () => now,
    })
    const first = npc({ id: 'npc-1', x: 0, y: 0, facing: 'left', moving: false })
    const second = npc({ id: 'npc-1', x: 100, y: 50, facing: 'right', moving: true })

    remoteNpcs.recordSnapshots([first], 1_000)
    remoteNpcs.recordSnapshots([second], 1_200)

    expect(remoteNpcs.smoothNpcs([second])).toEqual([{
      ...second,
      x: 50,
      y: 25,
      facing: 'right',
      moving: true,
    }])

    now = 1_400
    expect(remoteNpcs.smoothNpcs([second])[0]).toEqual(second)
  })

  it('replaces duplicate timestamp frames instead of growing history', () => {
    const remoteNpcs = useSunnyTownRemoteNpcs({
      interpolationDelayMs: 0,
      now: () => 1_000,
    })
    const first = npc({ id: 'npc-1', x: 10 })
    const replacement = npc({ id: 'npc-1', x: 20 })

    remoteNpcs.recordSnapshots([first], 1_000)
    remoteNpcs.recordSnapshots([replacement], 1_000)

    expect(remoteNpcs.smoothNpcs([first])[0]?.x).toBe(20)
  })

  it('drops NPC histories when an NPC disappears from authoritative snapshots', () => {
    const remoteNpcs = useSunnyTownRemoteNpcs({
      interpolationDelayMs: 50,
      now: () => 1_150,
    })
    const staleBefore = npc({ id: 'stale', x: 0 })
    const staleAfter = npc({ id: 'stale', x: 100 })
    const current = npc({ id: 'current', x: 30 })

    remoteNpcs.recordSnapshots([staleBefore], 1_000)
    remoteNpcs.recordSnapshots([staleAfter], 1_100)
    remoteNpcs.recordSnapshots([current], 1_150)

    expect(remoteNpcs.smoothNpcs([staleAfter])[0]).toEqual(staleAfter)
    expect(remoteNpcs.smoothNpcs([current])[0]).toEqual(current)
  })

  it('clears histories when an authoritative empty NPC snapshot arrives', () => {
    const remoteNpcs = useSunnyTownRemoteNpcs({
      interpolationDelayMs: 50,
      now: () => 1_150,
    })
    const first = npc({ id: 'npc-1', x: 0 })
    const second = npc({ id: 'npc-1', x: 100 })

    remoteNpcs.recordSnapshots([first], 1_000)
    remoteNpcs.recordSnapshots([second], 1_100)
    remoteNpcs.recordSnapshots([], 1_150)

    expect(remoteNpcs.smoothNpcs([])).toEqual([])
    expect(remoteNpcs.smoothNpcs([second])[0]).toEqual(second)
  })

  it('keeps only the configured number of history frames', () => {
    const remoteNpcs = useSunnyTownRemoteNpcs({
      interpolationDelayMs: 0,
      maxHistoryFrames: 2,
      now: () => 2_000,
    })
    const first = npc({ id: 'npc-1', x: 10 })
    const second = npc({ id: 'npc-1', x: 20 })
    const third = npc({ id: 'npc-1', x: 30 })

    remoteNpcs.recordSnapshots([first], 1_000)
    remoteNpcs.recordSnapshots([second], 1_100)
    remoteNpcs.recordSnapshots([third], 1_200)

    expect(remoteNpcs.smoothNpcs([third])[0]).toEqual(third)
    expect(interpolateRemoteNpc(third, [
      { at: 1_100, npc: second },
      { at: 1_200, npc: third },
    ], 1_050)).toEqual(second)
  })

  it('interpolates between frames and carries display fields from the later frame', () => {
    const before = npc({
      id: 'npc-1',
      x: 0,
      y: 0,
      facing: 'left',
      moving: true,
      name: 'Before',
      dialogue: ['before'],
    })
    const after = npc({
      id: 'npc-1',
      x: 40,
      y: 20,
      facing: 'right',
      moving: false,
      name: 'After',
      dialogue: ['after'],
      activity: { type: 'schoolwork' },
    })

    expect(interpolateRemoteNpc(after, [
      { at: 100, npc: before },
      { at: 300, npc: after },
    ], 200)).toEqual({
      ...after,
      x: 20,
      y: 10,
      facing: 'right',
      moving: true,
    })
  })

  it('returns the single recorded frame before there is enough history to interpolate', () => {
    const first = npc({ id: 'npc-1', x: 10 })
    const target = npc({ id: 'npc-1', x: 30 })

    expect(interpolateRemoteNpc(target, [{ at: 100, npc: first }], 200)).toEqual(first)
  })
})

function npc(overrides: Partial<SunnyTownNpc>): SunnyTownNpc {
  return {
    id: 'npc',
    name: 'NPC',
    x: 0,
    y: 0,
    facing: 'down',
    moving: false,
    spriteKey: 'default',
    dialogue: [],
    ...overrides,
  }
}
