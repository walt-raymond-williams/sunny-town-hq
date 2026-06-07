import { describe, expect, it } from 'vitest'
import { directionVector } from './characterDrawing'

describe('directionVector', () => {
  it('maps facings to unit vectors', () => {
    expect(directionVector('up')).toEqual({ x: 0, y: -1 })
    expect(directionVector('down')).toEqual({ x: 0, y: 1 })
    expect(directionVector('left')).toEqual({ x: -1, y: 0 })
    expect(directionVector('right')).toEqual({ x: 1, y: 0 })
  })
})
