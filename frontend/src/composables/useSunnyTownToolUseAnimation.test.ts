import { describe, expect, it } from 'vitest'
import { useSunnyTownToolUseAnimation } from './useSunnyTownToolUseAnimation'

describe('useSunnyTownToolUseAnimation', () => {
  it('reports clamped progress for the active tool', () => {
    const animation = useSunnyTownToolUseAnimation()

    animation.start('pickaxe', 'right', 100, 400)

    expect(animation.progress('pickaxe', 50)).toBe(0)
    expect(animation.progress('pickaxe', 300)).toBe(0.5)
  })

  it('ignores non-active tools', () => {
    const animation = useSunnyTownToolUseAnimation()

    animation.start('pickaxe', 'right', 100, 400)

    expect(animation.progress('shovel', 300)).toBeNull()
  })

  it('clears finished animations', () => {
    const animation = useSunnyTownToolUseAnimation()

    animation.start('pickaxe', 'right', 100, 400)

    expect(animation.progress('pickaxe', 500)).toBeNull()
    expect(animation.progress('pickaxe', 300)).toBeNull()
  })

  it('can be cleared manually', () => {
    const animation = useSunnyTownToolUseAnimation()

    animation.start('pickaxe', 'right', 100, 400)
    animation.clear()

    expect(animation.progress('pickaxe', 300)).toBeNull()
  })
})
