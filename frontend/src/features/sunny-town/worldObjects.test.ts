import { describe, expect, it } from 'vitest'
import type {
  SunnyTownPlacedObject,
  SunnyTownResourceNode,
  SunnyTownWorldObject,
} from '../../types/sunnyTown'
import {
  legacyWorldObjects,
  placedObjectToWorldObject,
  resourceNodeToWorldObject,
  sameWorldObject,
  worldObjectToPlacedObject,
  worldObjectToResourceNode,
} from './worldObjects'

describe('Sunny Town world object helpers', () => {
  it('converts resource nodes into natural world objects', () => {
    const node: SunnyTownResourceNode = {
      id: 'rock-1',
      kind: 'rock',
      x: 96,
      y: 128,
      radius: 20,
      active: true,
      hits: 1,
      needed: 3,
    }

    expect(resourceNodeToWorldObject(node)).toEqual({
      id: 'rock-1',
      kind: 'rock_node',
      source: 'natural',
      resourceKind: 'rock',
      x: 96,
      y: 128,
      radius: 20,
      active: true,
      collision: true,
      breakable: true,
      reservesPlacement: true,
      hits: 1,
      needed: 3,
    })
  })

  it('converts placed stone blocks into placed world objects', () => {
    const object: SunnyTownPlacedObject = {
      id: 'block-1',
      itemKey: 'stone_block',
      gridX: 4,
      gridY: 5,
      x: 128,
      y: 160,
      width: 32,
      height: 32,
      placedByAppUserId: 42,
    }

    expect(placedObjectToWorldObject(object)).toEqual({
      id: 'block-1',
      kind: 'stone_block',
      source: 'placed',
      itemKey: 'stone_block',
      x: 128,
      y: 160,
      width: 32,
      height: 32,
      active: true,
      collision: true,
      breakable: true,
      reservesPlacement: true,
      gridX: 4,
      gridY: 5,
      placedByAppUserId: 42,
    })
  })

  it('builds legacy world object lists with natural objects before placed objects', () => {
    const nodes: SunnyTownResourceNode[] = [{
      id: 'rock-1',
      kind: 'rock',
      x: 96,
      y: 128,
      radius: 20,
      active: true,
      hits: 0,
      needed: 3,
    }]
    const objects: SunnyTownPlacedObject[] = [{
      id: 'block-1',
      itemKey: 'stone_block',
      gridX: 4,
      gridY: 5,
      x: 128,
      y: 160,
      width: 32,
      height: 32,
    }]

    expect(legacyWorldObjects(nodes, objects).map((object) => object.source)).toEqual(['natural', 'placed'])
  })

  it('matches world object identity by source and id', () => {
    const first: SunnyTownWorldObject = {
      id: 'shared-id',
      kind: 'rock_node',
      source: 'natural',
      x: 0,
      y: 0,
      active: true,
      collision: true,
      breakable: true,
      reservesPlacement: true,
    }
    const same: SunnyTownWorldObject = { ...first, x: 10 }
    const differentSource: SunnyTownWorldObject = { ...first, kind: 'stone_block', source: 'placed' }

    expect(sameWorldObject(first, same)).toBe(true)
    expect(sameWorldObject(first, differentSource)).toBe(false)
  })

  it('converts world objects back into render-specific object shapes with defaults', () => {
    const placed: SunnyTownWorldObject = {
      id: 'block-1',
      kind: 'stone_block',
      source: 'placed',
      itemKey: 'stone_block',
      x: 128,
      y: 160,
      active: true,
      collision: true,
      breakable: true,
      reservesPlacement: true,
    }
    const natural: SunnyTownWorldObject = {
      id: 'rock-1',
      kind: 'rock_node',
      source: 'natural',
      resourceKind: 'rock',
      x: 96,
      y: 128,
      active: false,
      collision: true,
      breakable: true,
      reservesPlacement: true,
    }

    expect(worldObjectToPlacedObject(placed)).toEqual({
      id: 'block-1',
      itemKey: 'stone_block',
      gridX: 0,
      gridY: 0,
      x: 128,
      y: 160,
      width: 0,
      height: 0,
      placedByAppUserId: undefined,
    })
    expect(worldObjectToResourceNode(natural)).toEqual({
      id: 'rock-1',
      kind: 'rock',
      x: 96,
      y: 128,
      radius: 0,
      active: false,
      hits: 0,
      needed: 0,
    })
  })
})
