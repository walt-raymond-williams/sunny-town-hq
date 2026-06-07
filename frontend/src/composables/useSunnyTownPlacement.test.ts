import { describe, expect, it } from 'vitest'
import type { SunnyTownMap, SunnyTownPlayer, SunnyTownWorldObject } from '../types/sunnyTown'
import { gridFromPointerPosition, useSunnyTownPlacement } from './useSunnyTownPlacement'

describe('useSunnyTownPlacement', () => {
  it('converts pointer coordinates into map grid coordinates using camera offset', () => {
    expect(gridFromPointerPosition(
      map(),
      { left: 10, top: 20, width: 320, height: 320 },
      { x: 64, y: 32 },
      75,
      91,
    )).toEqual({ gridX: 4, gridY: 3 })
  })

  it('returns null for pointer positions outside the map', () => {
    expect(gridFromPointerPosition(
      map(),
      { left: 0, top: 0, width: 320, height: 320 },
      { x: 0, y: 0 },
      -1,
      20,
    )).toBeNull()
    expect(gridFromPointerPosition(
      map(),
      { left: 0, top: 0, width: 320, height: 320 },
      { x: 0, y: 0 },
      321,
      20,
    )).toBeNull()
  })

  it('stores and clears hover grid state', () => {
    const placement = useSunnyTownPlacement()
    const canvas = {
      getBoundingClientRect: () => ({ left: 0, top: 0, width: 320, height: 320 }),
    } as HTMLCanvasElement

    placement.setHoverFromPointer({ clientX: 50, clientY: 70 } as PointerEvent, map(), canvas, { x: 0, y: 0 })
    expect(placement.hoverGrid.value).toEqual({ gridX: 1, gridY: 2 })

    placement.clearHover()
    expect(placement.hoverGrid.value).toBeNull()
  })

  it('checks stone block placement against quantity and map collisions', () => {
    const placement = useSunnyTownPlacement()
    const self = player({ x: 200, y: 200 })
    const blockingObject: SunnyTownWorldObject = {
      id: 'blocker',
      kind: 'stone_block',
      source: 'placed',
      itemKey: 'stone_block',
      x: 32,
      y: 32,
      width: 32,
      height: 32,
      active: true,
      collision: true,
      breakable: true,
      reservesPlacement: true,
    }

    expect(placement.canPlaceStoneBlock(map(), 2, 2, 1, [], self)).toBe(true)
    expect(placement.canPlaceStoneBlock(map(), 2, 2, 0, [], self)).toBe(false)
    expect(placement.canPlaceStoneBlock(map(), 1, 1, 1, [blockingObject], self)).toBe(false)
  })
})

function map(): SunnyTownMap {
  return {
    id: 'test-map',
    name: 'Test Map',
    tileSize: 32,
    width: 10,
    height: 10,
    spawns: [],
    blockedRects: [],
    starSpawns: [],
    portals: [],
    npcs: [],
    resourceNodes: [],
  }
}

function player(overrides: Partial<SunnyTownPlayer>): SunnyTownPlayer {
  return {
    id: 'self',
    displayName: 'Self',
    x: 0,
    y: 0,
    facing: 'down',
    moving: false,
    avatarId: 'default',
    lastProcessedSeq: 0,
    ...overrides,
  }
}
