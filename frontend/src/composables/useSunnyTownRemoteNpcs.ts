import type { SunnyTownNpc } from '../types/sunnyTown'

export interface RemoteNpcHistoryFrame {
  at: number
  npc: SunnyTownNpc
}

interface SunnyTownRemoteNpcsOptions {
  interpolationDelayMs?: number
  maxHistoryFrames?: number
  now?: () => number
}

const defaultInterpolationDelayMs = 150
const defaultMaxHistoryFrames = 12

export function useSunnyTownRemoteNpcs(options: SunnyTownRemoteNpcsOptions = {}) {
  const interpolationDelayMs = options.interpolationDelayMs ?? defaultInterpolationDelayMs
  const maxHistoryFrames = options.maxHistoryFrames ?? defaultMaxHistoryFrames
  const now = options.now ?? Date.now
  const histories = new Map<string, RemoteNpcHistoryFrame[]>()

  function clear() {
    histories.clear()
  }

  function recordSnapshots(snapshotNpcs: SunnyTownNpc[], snapshotAt: number) {
    const npcIds = new Set<string>()
    for (const npc of snapshotNpcs) {
      npcIds.add(npc.id)
      const history = histories.get(npc.id) || []
      const latest = history[history.length - 1]
      if (latest && latest.at === snapshotAt) {
        latest.npc = { ...npc }
        continue
      }

      history.push({ at: snapshotAt, npc: { ...npc } })
      while (history.length > maxHistoryFrames) {
        history.shift()
      }
      histories.set(npc.id, history)
    }

    for (const npcId of histories.keys()) {
      if (!npcIds.has(npcId)) {
        histories.delete(npcId)
      }
    }
  }

  function smoothNpcs(npcs: SunnyTownNpc[]): SunnyTownNpc[] {
    const renderAt = now() - interpolationDelayMs
    return npcs.map((target) => interpolateRemoteNpc(target, histories.get(target.id), renderAt))
  }

  return {
    clear,
    recordSnapshots,
    smoothNpcs,
  }
}

export function interpolateRemoteNpc(
  target: SunnyTownNpc,
  history: RemoteNpcHistoryFrame[] | undefined,
  renderAt: number,
): SunnyTownNpc {
  if (!history || history.length === 0) {
    return { ...target }
  }
  const first = history[0]
  const latest = history[history.length - 1]
  if (!first || !latest) {
    return { ...target }
  }
  if (history.length === 1 || renderAt <= first.at) {
    return { ...first.npc }
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
    return { ...after.npc }
  }

  const progress = clamp((renderAt - before.at) / (after.at - before.at), 0, 1)
  return {
    ...after.npc,
    x: before.npc.x + (after.npc.x - before.npc.x) * progress,
    y: before.npc.y + (after.npc.y - before.npc.y) * progress,
    facing: progress < 0.5 ? before.npc.facing : after.npc.facing,
    moving: before.npc.moving || after.npc.moving,
  }
}

function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value))
}
