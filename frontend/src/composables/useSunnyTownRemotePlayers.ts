import type { SunnyTownPlayer } from '../types/sunnyTown'

export interface RemotePlayerHistoryFrame {
  at: number
  player: SunnyTownPlayer
}

interface SunnyTownRemotePlayersOptions {
  interpolationDelayMs?: number
  maxHistoryFrames?: number
  now?: () => number
}

const defaultInterpolationDelayMs = 150
const defaultMaxHistoryFrames = 12

export function useSunnyTownRemotePlayers(options: SunnyTownRemotePlayersOptions = {}) {
  const interpolationDelayMs = options.interpolationDelayMs ?? defaultInterpolationDelayMs
  const maxHistoryFrames = options.maxHistoryFrames ?? defaultMaxHistoryFrames
  const now = options.now ?? Date.now
  const histories = new Map<string, RemotePlayerHistoryFrame[]>()

  function clear() {
    histories.clear()
  }

  function recordSnapshots(snapshotPlayers: SunnyTownPlayer[], selfId: string, snapshotAt: number) {
    const remoteIds = new Set<string>()
    for (const player of snapshotPlayers) {
      if (player.id === selfId) {
        continue
      }

      remoteIds.add(player.id)
      const history = histories.get(player.id) || []
      const latest = history[history.length - 1]
      if (latest && latest.at === snapshotAt) {
        latest.player = { ...player }
        continue
      }

      history.push({ at: snapshotAt, player: { ...player } })
      while (history.length > maxHistoryFrames) {
        history.shift()
      }
      histories.set(player.id, history)
    }

    for (const playerId of histories.keys()) {
      if (!remoteIds.has(playerId)) {
        histories.delete(playerId)
      }
    }
  }

  function smoothPlayers(remotePlayers: SunnyTownPlayer[]): SunnyTownPlayer[] {
    const renderAt = now() - interpolationDelayMs
    return remotePlayers.map((target) => interpolateRemotePlayer(target, histories.get(target.id), renderAt))
  }

  return {
    clear,
    recordSnapshots,
    smoothPlayers,
  }
}

export function interpolateRemotePlayer(
  target: SunnyTownPlayer,
  history: RemotePlayerHistoryFrame[] | undefined,
  renderAt: number,
): SunnyTownPlayer {
  if (!history || history.length === 0) {
    return { ...target }
  }
  const first = history[0]
  const latest = history[history.length - 1]
  if (!first || !latest) {
    return { ...target }
  }
  if (history.length === 1 || renderAt <= first.at) {
    return { ...first.player }
  }

  let before = first
  let after = latest
  for (let index = 1; index < history.length; index++) {
    const candidate = history[index]
    if (!candidate) {
      continue
    }
    if (candidate.at >= renderAt) {
      after = candidate
      break
    }
    before = candidate
  }

  if (renderAt >= after.at || after.at <= before.at) {
    return { ...after.player }
  }

  const progress = clamp((renderAt - before.at) / (after.at - before.at), 0, 1)
  return {
    ...after.player,
    x: before.player.x + (after.player.x - before.player.x) * progress,
    y: before.player.y + (after.player.y - before.player.y) * progress,
    facing: progress < 0.5 ? before.player.facing : after.player.facing,
    moving: before.player.moving || after.player.moving,
  }
}

function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value))
}
