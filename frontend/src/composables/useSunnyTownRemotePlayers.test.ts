import { describe, expect, it } from 'vitest'
import type { SunnyTownPlayer } from '../types/sunnyTown'
import { interpolateRemotePlayer, useSunnyTownRemotePlayers } from './useSunnyTownRemotePlayers'

describe('useSunnyTownRemotePlayers', () => {
  it('records only remote players and smooths them at the delayed render time', () => {
    let now = 1_250
    const remotePlayers = useSunnyTownRemotePlayers({
      interpolationDelayMs: 150,
      now: () => now,
    })
    const self = player({ id: 'self', x: 1, y: 1 })
    const firstRemote = player({ id: 'remote', x: 0, y: 0, facing: 'left', moving: false })
    const secondRemote = player({ id: 'remote', x: 100, y: 50, facing: 'right', moving: true })

    remotePlayers.recordSnapshots([self, firstRemote], 'self', 1_000)
    remotePlayers.recordSnapshots([self, secondRemote], 'self', 1_200)

    expect(remotePlayers.smoothPlayers([secondRemote])).toEqual([{
      ...secondRemote,
      x: 50,
      y: 25,
      facing: 'right',
      moving: true,
    }])

    now = 1_400
    expect(remotePlayers.smoothPlayers([secondRemote])[0]).toEqual(secondRemote)
  })

  it('replaces duplicate timestamp frames instead of growing history', () => {
    const remotePlayers = useSunnyTownRemotePlayers({
      interpolationDelayMs: 0,
      now: () => 1_000,
    })
    const first = player({ id: 'remote', x: 10 })
    const replacement = player({ id: 'remote', x: 20 })

    remotePlayers.recordSnapshots([first], 'self', 1_000)
    remotePlayers.recordSnapshots([replacement], 'self', 1_000)

    expect(remotePlayers.smoothPlayers([first])[0]?.x).toBe(20)
  })

  it('drops remote histories when a player disappears from snapshots', () => {
    const remotePlayers = useSunnyTownRemotePlayers({
      interpolationDelayMs: 0,
      now: () => 1_000,
    })
    const stale = player({ id: 'stale', x: 10 })
    const current = player({ id: 'current', x: 30 })

    remotePlayers.recordSnapshots([stale], 'self', 900)
    remotePlayers.recordSnapshots([current], 'self', 1_000)

    expect(remotePlayers.smoothPlayers([stale])[0]).toEqual(stale)
    expect(remotePlayers.smoothPlayers([current])[0]).toEqual(current)
  })

  it('keeps only the configured number of history frames', () => {
    const remotePlayers = useSunnyTownRemotePlayers({
      interpolationDelayMs: 0,
      maxHistoryFrames: 2,
      now: () => 2_000,
    })
    const first = player({ id: 'remote', x: 10 })
    const second = player({ id: 'remote', x: 20 })
    const third = player({ id: 'remote', x: 30 })

    remotePlayers.recordSnapshots([first], 'self', 1_000)
    remotePlayers.recordSnapshots([second], 'self', 1_100)
    remotePlayers.recordSnapshots([third], 'self', 1_200)

    expect(remotePlayers.smoothPlayers([third])[0]).toEqual(third)
    expect(interpolateRemotePlayer(third, [
      { at: 1_100, player: second },
      { at: 1_200, player: third },
    ], 1_050)).toEqual(second)
  })

  it('interpolates between frames and carries stable display fields from the later frame', () => {
    const before = player({ id: 'remote', x: 0, y: 0, displayName: 'Before', facing: 'left', moving: true })
    const after = player({ id: 'remote', x: 40, y: 20, displayName: 'After', facing: 'right', moving: false })

    expect(interpolateRemotePlayer(after, [
      { at: 100, player: before },
      { at: 300, player: after },
    ], 200)).toEqual({
      ...after,
      x: 20,
      y: 10,
      facing: 'right',
      moving: true,
    })
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
