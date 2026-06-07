import type {
  SunnyTownPlacedObject,
  SunnyTownResourceNode,
  SunnyTownWorldObject,
} from '../../types/sunnyTown'

export function legacyWorldObjects(
  nodes: SunnyTownResourceNode[],
  objects: SunnyTownPlacedObject[],
): SunnyTownWorldObject[] {
  return [
    ...nodes.map(resourceNodeToWorldObject),
    ...objects.map(placedObjectToWorldObject),
  ]
}

export function resourceNodeToWorldObject(node: SunnyTownResourceNode): SunnyTownWorldObject {
  return {
    id: node.id,
    kind: 'rock_node',
    source: 'natural',
    resourceKind: node.kind,
    x: node.x,
    y: node.y,
    radius: node.radius,
    active: node.active,
    collision: true,
    breakable: true,
    reservesPlacement: true,
    hits: node.hits,
    needed: node.needed,
  }
}

export function placedObjectToWorldObject(object: SunnyTownPlacedObject): SunnyTownWorldObject {
  return {
    id: object.id,
    kind: 'stone_block',
    source: 'placed',
    itemKey: object.itemKey,
    x: object.x,
    y: object.y,
    width: object.width,
    height: object.height,
    active: true,
    collision: true,
    breakable: true,
    reservesPlacement: true,
    gridX: object.gridX,
    gridY: object.gridY,
    placedByAppUserId: object.placedByAppUserId,
  }
}

export function sameWorldObject(first: SunnyTownWorldObject, second: SunnyTownWorldObject): boolean {
  return first.source === second.source && first.id === second.id
}

export function worldObjectToPlacedObject(object: SunnyTownWorldObject): SunnyTownPlacedObject {
  return {
    id: object.id,
    itemKey: 'stone_block',
    gridX: object.gridX || 0,
    gridY: object.gridY || 0,
    x: object.x,
    y: object.y,
    width: object.width || 0,
    height: object.height || 0,
    placedByAppUserId: object.placedByAppUserId,
  }
}

export function worldObjectToResourceNode(object: SunnyTownWorldObject): SunnyTownResourceNode {
  return {
    id: object.id,
    kind: 'rock',
    x: object.x,
    y: object.y,
    radius: object.radius || 0,
    active: object.active,
    hits: object.hits || 0,
    needed: object.needed || 0,
  }
}
