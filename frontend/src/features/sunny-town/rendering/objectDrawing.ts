import type {
  SunnyTownMap,
  SunnyTownPlacedObject,
  SunnyTownResourceNode,
  SunnyTownWorldObject,
} from '../../../types/sunnyTown'
import {
  worldObjectToPlacedObject,
  worldObjectToResourceNode,
} from '../worldObjects'

export interface PlacementPreviewGrid {
  gridX: number
  gridY: number
}

export function drawWorldObjects(
  context: CanvasRenderingContext2D,
  objects: SunnyTownWorldObject[],
  cameraX: number,
  cameraY: number,
) {
  for (const object of objects) {
    if (!object.active) {
      continue
    }
    if (object.kind === 'stone_block' && object.itemKey === 'stone_block') {
      drawPlacedObject(context, worldObjectToPlacedObject(object), cameraX, cameraY)
      continue
    }
    if (object.kind === 'rock_node' && object.resourceKind === 'rock') {
      drawResourceNode(context, worldObjectToResourceNode(object), cameraX, cameraY)
      continue
    }
    if (object.kind === 'chest') {
      drawChestObject(context, object, cameraX, cameraY)
    }
  }
}

export function drawChestObject(
  context: CanvasRenderingContext2D,
  object: SunnyTownWorldObject,
  cameraX: number,
  cameraY: number,
) {
  const width = object.width || 32
  const height = object.height || 32
  const x = object.x - cameraX
  const y = object.y - cameraY
  context.save()
  context.fillStyle = '#8a5a35'
  context.fillRect(x + 3, y + 8, width - 6, height - 11)
  context.fillStyle = '#a97844'
  context.fillRect(x + 5, y + 5, width - 10, 10)
  context.strokeStyle = '#4f3320'
  context.lineWidth = 2
  context.strokeRect(x + 3, y + 8, width - 6, height - 11)
  context.strokeRect(x + 5, y + 5, width - 10, 10)
  context.fillStyle = '#f2c14e'
  context.fillRect(x + width / 2 - 3, y + height / 2 - 1, 6, 7)
  context.strokeStyle = '#6f4a24'
  context.beginPath()
  context.moveTo(x + 6, y + height / 2)
  context.lineTo(x + width - 6, y + height / 2)
  context.stroke()
  context.restore()
}

export function drawPlacedObject(
  context: CanvasRenderingContext2D,
  object: SunnyTownPlacedObject,
  cameraX: number,
  cameraY: number,
) {
  const x = object.x - cameraX
  const y = object.y - cameraY
  context.save()
  context.fillStyle = '#69727c'
  context.fillRect(x + 3, y + 3, object.width - 6, object.height - 6)
  context.strokeStyle = '#353b42'
  context.lineWidth = 2
  context.strokeRect(x + 3, y + 3, object.width - 6, object.height - 6)
  context.fillStyle = '#8d98a3'
  context.fillRect(x + 7, y + 7, object.width - 14, 5)
  context.fillStyle = '#4d555e'
  context.fillRect(x + 7, y + object.height - 12, object.width - 14, 4)
  context.restore()
}

export function drawPlacementPreview(
  context: CanvasRenderingContext2D,
  map: SunnyTownMap,
  cameraX: number,
  cameraY: number,
  hoverGrid: PlacementPreviewGrid | null,
  valid: boolean,
) {
  if (!hoverGrid) {
    return
  }
  const { gridX, gridY } = hoverGrid
  const x = gridX * map.tileSize - cameraX
  const y = gridY * map.tileSize - cameraY
  context.save()
  context.globalAlpha = 0.72
  context.fillStyle = valid ? '#8d98a3' : '#b94a48'
  context.fillRect(x + 3, y + 3, map.tileSize - 6, map.tileSize - 6)
  context.globalAlpha = 1
  context.strokeStyle = valid ? '#f7e08a' : '#ffcbc7'
  context.lineWidth = 2
  context.strokeRect(x + 2, y + 2, map.tileSize - 4, map.tileSize - 4)
  context.restore()
}

export function drawResourceNode(
  context: CanvasRenderingContext2D,
  node: SunnyTownResourceNode,
  cameraX: number,
  cameraY: number,
) {
  if (!node.active) {
    return
  }
  const x = node.x - cameraX
  const y = node.y - cameraY
  context.save()
  context.translate(x, y)
  context.fillStyle = '#6b737b'
  context.strokeStyle = '#343a40'
  context.lineWidth = 3
  context.beginPath()
  context.moveTo(-node.radius, 4)
  context.lineTo(-node.radius * 0.55, -node.radius * 0.7)
  context.lineTo(node.radius * 0.25, -node.radius)
  context.lineTo(node.radius, -node.radius * 0.1)
  context.lineTo(node.radius * 0.7, node.radius * 0.75)
  context.lineTo(-node.radius * 0.45, node.radius)
  context.closePath()
  context.fill()
  context.stroke()
  context.fillStyle = '#bcd5e8'
  context.beginPath()
  context.arc(node.radius * 0.25, -node.radius * 0.35, 4, 0, Math.PI * 2)
  context.fill()
  const hits = Math.max(0, node.hits || 0)
  if (hits > 0) {
    context.strokeStyle = '#23282e'
    context.lineWidth = 2
    context.beginPath()
    context.moveTo(-node.radius * 0.15, -node.radius * 0.75)
    context.lineTo(node.radius * 0.05, -node.radius * 0.25)
    context.lineTo(-node.radius * 0.2, node.radius * 0.15)
    context.stroke()
  }
  if (hits > 1) {
    context.beginPath()
    context.moveTo(node.radius * 0.2, -node.radius * 0.45)
    context.lineTo(node.radius * 0.45, -node.radius * 0.05)
    context.lineTo(node.radius * 0.25, node.radius * 0.45)
    context.stroke()
  }
  context.restore()
}
