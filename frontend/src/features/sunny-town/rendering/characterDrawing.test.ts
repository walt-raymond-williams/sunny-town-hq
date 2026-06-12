import { describe, expect, it } from 'vitest'
import { directionVector, npcRoutineCue } from './characterDrawing'

describe('directionVector', () => {
  it('maps facings to unit vectors', () => {
    expect(directionVector('up')).toEqual({ x: 0, y: -1 })
    expect(directionVector('down')).toEqual({ x: 0, y: 1 })
    expect(directionVector('left')).toEqual({ x: -1, y: 0 })
    expect(directionVector('right')).toEqual({ x: 1, y: 0 })
  })
})

describe('npcRoutineCue', () => {
  it('maps authoritative routine statuses to compact cue symbols', () => {
    expect(npcRoutineCue('traveling')?.symbol).toBe('>')
    expect(npcRoutineCue('resting')?.symbol).toBe('Z')
    expect(npcRoutineCue('working')?.symbol).toBe('W')
    expect(npcRoutineCue('blocked')?.symbol).toBe('!')
  })

  it('omits routine cues when no trustworthy status is present', () => {
    expect(npcRoutineCue(undefined)).toBeNull()
  })
})
