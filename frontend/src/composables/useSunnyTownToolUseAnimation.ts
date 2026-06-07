import type { SunnyTownPlayer } from '../types/sunnyTown'

interface ToolUseAnimation {
  toolKey: string
  startedAt: number
  durationMs: number
  facing: SunnyTownPlayer['facing']
}

export function useSunnyTownToolUseAnimation() {
  let activeToolUse: ToolUseAnimation | null = null

  function clear() {
    activeToolUse = null
  }

  function start(
    toolKey: string,
    facing: SunnyTownPlayer['facing'],
    startedAt: number,
    durationMs: number,
  ) {
    activeToolUse = {
      toolKey,
      startedAt,
      durationMs,
      facing,
    }
  }

  function progress(toolKey: string, now: number): number | null {
    if (!activeToolUse || activeToolUse.toolKey !== toolKey) {
      return null
    }
    const nextProgress = (now - activeToolUse.startedAt) / activeToolUse.durationMs
    if (nextProgress >= 1) {
      activeToolUse = null
      return null
    }
    return clamp(nextProgress, 0, 1)
  }

  return {
    clear,
    progress,
    start,
  }
}

function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value))
}
