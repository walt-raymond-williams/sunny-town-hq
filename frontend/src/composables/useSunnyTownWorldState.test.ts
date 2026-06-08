import { describe, expect, it } from 'vitest'
import type {
  SunnyTownMap,
  SunnyTownNpc,
  SunnyTownPlacedObject,
  SunnyTownResourceNode,
  SunnyTownServerMessage,
  SunnyTownWorldObject,
} from '../types/sunnyTown'
import {
  normalizeSunnyTownMap,
  useSunnyTownWorldState,
} from './useSunnyTownWorldState'

function map(overrides: Partial<SunnyTownMap> = {}): SunnyTownMap {
  return {
    id: 'sunny-town-v1',
    name: 'Sunny Town',
    tileSize: 32,
    width: 40,
    height: 30,
    spawns: [],
    blockedRects: [],
    starSpawns: [],
    portals: [],
    npcs: [],
    resourceNodes: [],
    ...overrides,
  }
}

function placedObject(overrides: Partial<SunnyTownPlacedObject> = {}): SunnyTownPlacedObject {
  return {
    id: 'block-1',
    itemKey: 'stone_block',
    gridX: 4,
    gridY: 5,
    x: 128,
    y: 160,
    width: 32,
    height: 32,
    ...overrides,
  }
}

function npc(overrides: Partial<SunnyTownNpc> = {}): SunnyTownNpc {
  return {
    id: 'guide',
    name: 'Guide',
    x: 160,
    y: 160,
    facing: 'down',
    spriteKey: 'guide',
    dialogue: ['Hello.'],
    ...overrides,
  }
}

function rockNode(overrides: Partial<SunnyTownResourceNode> = {}): SunnyTownResourceNode {
  return {
    id: 'rock-1',
    kind: 'rock',
    x: 96,
    y: 128,
    radius: 20,
    active: true,
    hits: 0,
    needed: 3,
    ...overrides,
  }
}

describe('normalizeSunnyTownMap', () => {
  it('fills optional map arrays from sparse server payloads', () => {
    const sparse = {
      id: 'forest-crossing-v1',
      name: 'Forest Crossing',
      tileSize: 32,
      width: 30,
      height: 30,
    } as SunnyTownMap

    expect(normalizeSunnyTownMap(sparse)).toEqual({
      ...sparse,
      blockedRects: [],
      fixtures: [],
      npcs: [],
      portals: [],
      resourceNodes: [],
      spawns: [],
      starSpawns: [],
    })
  })
})

describe('useSunnyTownWorldState', () => {
  it('applies map state and builds legacy world objects when needed', () => {
    const state = useSunnyTownWorldState()
    const object = placedObject()
    const node = rockNode()

    state.applyMapState({
      type: 'hello',
      map: map({ npcs: [npc()] }),
      players: [{ id: 'self', displayName: 'Ari', x: 1, y: 2, facing: 'down', moving: false, avatarId: 'sunny', lastProcessedSeq: 0 }],
      npcs: [npc({ x: 192 })],
      collectibles: [{ id: 'star-1', kind: 'star', x: 10, y: 12, active: true }],
      resourceNodes: [node],
      placedObjects: [object],
    })

    expect(state.activeMap.value?.id).toBe('sunny-town-v1')
    expect(state.players.value).toHaveLength(1)
    expect(state.npcs.value[0]?.x).toBe(192)
    expect(state.collectibles.value).toHaveLength(1)
    expect(state.worldObjects.value.map((worldObject) => worldObject.source)).toEqual(['natural', 'placed'])
  })

  it('falls back to static map npcs when map state omits live npcs', () => {
    const state = useSunnyTownWorldState()

    state.applyMapState({
      type: 'hello',
      map: map({ npcs: [npc()] }),
    })

    expect(state.npcs.value.map((liveNpc) => liveNpc.id)).toEqual(['guide'])
  })

  it('applies live npc snapshots for the active map', () => {
    const state = useSunnyTownWorldState()
    state.applyMapState({
      type: 'hello',
      map: map({ npcs: [npc()] }),
    })

    const applied = state.applySnapshot({
      type: 'snapshot',
      mapId: 'sunny-town-v1',
      npcs: [npc({ x: 220, y: 180, moving: true })],
    })

    expect(applied).toBe(true)
    expect(state.npcs.value[0]).toMatchObject({ id: 'guide', x: 220, y: 180, moving: true })
  })

  it('ignores snapshots from a stale map', () => {
    const state = useSunnyTownWorldState()
    state.applyMapState({ type: 'hello', map: map({ id: 'sunny-town-v1' }) })

    const applied = state.applySnapshot({
      type: 'snapshot',
      mapId: 'forest-crossing-v1',
      players: [{ id: 'remote', displayName: 'Bea', x: 1, y: 2, facing: 'down', moving: false, avatarId: 'sunny', lastProcessedSeq: 0 }],
    })

    expect(applied).toBe(false)
    expect(state.players.value).toEqual([])
  })

  it('preserves placed objects on snapshots that omit them', () => {
    const state = useSunnyTownWorldState()
    state.applyMapState({
      type: 'hello',
      map: map(),
      placedObjects: [placedObject()],
    })

    state.applySnapshot({
      type: 'snapshot',
      mapId: 'sunny-town-v1',
      resourceNodes: [rockNode()],
    })

    expect(state.placedObjects.value.map((object) => object.id)).toEqual(['block-1'])
    expect(state.worldObjects.value.map((object) => object.id)).toEqual(['rock-1', 'block-1'])
  })

  it('upserts placed object messages into placed and world object state', () => {
    const state = useSunnyTownWorldState()
    const object = placedObject({ id: 'block-1', x: 128 })
    state.applyMapState({ type: 'hello', map: map(), placedObjects: [object] })

    state.applyPlacedObject({
      type: 'map_object_placed',
      placedObject: placedObject({ id: 'block-1', x: 160 }),
    } as SunnyTownServerMessage)

    expect(state.placedObjects.value).toHaveLength(1)
    expect(state.placedObjects.value[0]?.x).toBe(160)
    expect(state.worldObjects.value.find((worldObject) => worldObject.id === 'block-1')?.x).toBe(160)
  })

  it('removes placed object messages by explicit world object identity', () => {
    const state = useSunnyTownWorldState()
    const natural: SunnyTownWorldObject = {
      id: 'shared',
      kind: 'rock_node',
      source: 'natural',
      x: 32,
      y: 32,
      active: true,
      collision: true,
      breakable: true,
      reservesPlacement: true,
    }
    state.applyMapState({
      type: 'hello',
      map: map(),
      worldObjects: [
        natural,
        { ...natural, kind: 'stone_block', source: 'placed' },
      ],
    })

    state.applyRemovedObject({
      type: 'map_object_removed',
      worldObject: { ...natural, kind: 'stone_block', source: 'placed' },
    } as SunnyTownServerMessage)

    expect(state.worldObjects.value).toEqual([natural])
  })
})
