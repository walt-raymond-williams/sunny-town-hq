import type { SunnyTownMap } from '../../../types/sunnyTown'

export function drawMap(
  context: CanvasRenderingContext2D,
  map: SunnyTownMap,
  cameraX: number,
  cameraY: number,
  width: number,
  height: number,
) {
  context.fillStyle = map.id === 'sunny-town-v1'
    ? '#8fcf85'
    : map.id === 'sunny-town-classroom' ? '#b7c6da' : map.id === 'forest-crossing-v1' ? '#6fb27a' : '#cda66f'
  context.fillRect(0, 0, width, height)

  if (map.id === 'sunny-town-v1') {
    context.fillStyle = '#d6bd79'
    context.fillRect(0 - cameraX, 420 - cameraY, map.width * map.tileSize, 124)
    context.fillRect(570 - cameraX, 0 - cameraY, 140, map.height * map.tileSize)
  } else if (map.id === 'sunny-town-classroom') {
    context.fillStyle = '#e4d4b5'
    context.fillRect(64 - cameraX, 64 - cameraY, map.width * map.tileSize - 128, map.height * map.tileSize - 128)
    context.fillStyle = '#3f596f'
    context.fillRect(224 - cameraX, 82 - cameraY, 192, 44)
  } else if (map.id === 'forest-crossing-v1') {
    context.fillStyle = '#7ac27d'
    context.fillRect(0, 0, width, height)
    context.fillStyle = '#d1b06a'
    context.fillRect(0 - cameraX, 420 - cameraY, map.width * map.tileSize, 124)
    context.fillStyle = '#3b88a3'
    context.fillRect(560 - cameraX, 0 - cameraY, 96, 384)
    context.fillRect(560 - cameraX, 576 - cameraY, 96, 384)
    context.fillStyle = '#a87c42'
    context.fillRect(548 - cameraX, 384 - cameraY, 120, 192)
    context.strokeStyle = '#765631'
    context.lineWidth = 4
    context.strokeRect(548 - cameraX, 384 - cameraY, 120, 192)
  } else {
    context.fillStyle = '#d9bd8d'
    context.fillRect(64 - cameraX, 64 - cameraY, map.width * map.tileSize - 128, map.height * map.tileSize - 128)
  }

  context.strokeStyle = 'rgba(255, 255, 255, 0.18)'
  context.lineWidth = 1
  for (let x = -cameraX % map.tileSize; x < width; x += map.tileSize) {
    context.beginPath()
    context.moveTo(x, 0)
    context.lineTo(x, height)
    context.stroke()
  }
  for (let y = -cameraY % map.tileSize; y < height; y += map.tileSize) {
    context.beginPath()
    context.moveTo(0, y)
    context.lineTo(width, y)
    context.stroke()
  }

  for (const blocked of map.blockedRects) {
    context.fillStyle = map.id === 'sunny-town-v1'
      ? blocked.width > 400 || blocked.height > 400 ? '#4f8a5b' : '#7e6b52'
      : map.id === 'sunny-town-classroom'
        ? blocked.width > 260 || blocked.height > 260 ? '#516070' : '#8a6f4d'
        : map.id === 'forest-crossing-v1'
          ? blocked.width > 90 || blocked.height > 90 ? '#3e7a45' : '#6b6f57'
          : blocked.width > 260 || blocked.height > 260 ? '#6d4f38' : '#8b6748'
    context.fillRect(blocked.x - cameraX, blocked.y - cameraY, blocked.width, blocked.height)
    if (map.id === 'forest-crossing-v1' && blocked.width <= 160 && blocked.height <= 160) {
      context.fillStyle = '#2f6b3b'
      context.beginPath()
      context.arc(blocked.x + blocked.width / 2 - cameraX, blocked.y + blocked.height / 2 - cameraY, Math.min(blocked.width, blocked.height) / 2, 0, Math.PI * 2)
      context.fill()
    }
  }

  for (const portal of map.portals) {
    context.fillStyle = '#3d2c22'
    context.fillRect(portal.x - cameraX, portal.y - cameraY, portal.width, portal.height)
    context.strokeStyle = '#f1d28f'
    context.lineWidth = 2
    context.strokeRect(portal.x - cameraX + 2, portal.y - cameraY + 2, portal.width - 4, portal.height - 4)
  }
}
