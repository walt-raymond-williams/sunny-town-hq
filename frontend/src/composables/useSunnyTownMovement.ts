import type { SunnyTownMap, SunnyTownPlayer, SunnyTownWorldObject } from '../types/sunnyTown'

export interface MovementInput {
  up: boolean
  down: boolean
  left: boolean
  right: boolean
}

export type MovementDirection = keyof MovementInput

export const playerSpeed = 150
export const playerSize = 28
export const moveSendIntervalMs = 50
export const movementInputEventOptions = { capture: true }

interface Rect {
  x: number
  y: number
  width: number
  height: number
}

export function useSunnyTownMovement() {
  const pressedDirections = new Set<MovementDirection>()

  function press(direction: MovementDirection) {
    pressedDirections.add(direction)
  }

  function release(direction: MovementDirection) {
    pressedDirections.delete(direction)
  }

  function clear() {
    pressedDirections.clear()
  }

  function hasInput(): boolean {
    return pressedDirections.size > 0
  }

  function currentInput(): MovementInput {
    return {
      up: pressedDirections.has('up'),
      down: pressedDirections.has('down'),
      left: pressedDirections.has('left'),
      right: pressedDirections.has('right'),
    }
  }

  return {
    clear,
    currentInput,
    hasInput,
    press,
    release,
  }
}

export function simulatePlayer(
  player: SunnyTownPlayer,
  input: MovementInput,
  map: SunnyTownMap | null,
  worldObjects: SunnyTownWorldObject[],
  deltaSeconds: number,
): SunnyTownPlayer {
  if (!map) {
    return player
  }
  const result = { ...player }
  const vector = movementVector(input)
  if (!vector) {
    result.moving = false
    return result
  }

  const nextX = result.x + vector.x * playerSpeed * deltaSeconds
  const nextY = result.y + vector.y * playerSpeed * deltaSeconds
  if (!collides(map, worldObjects, nextX, result.y)) {
    result.x = clampPlayerX(map, nextX)
  }
  if (!collides(map, worldObjects, result.x, nextY)) {
    result.y = clampPlayerY(map, nextY)
  }
  result.facing = movementFacing(vector)
  result.moving = true
  return result
}

export function canPlaceStoneBlock(
  map: SunnyTownMap,
  gridX: number,
  gridY: number,
  stoneBlockQuantity: number,
  worldObjects: SunnyTownWorldObject[],
  self: SunnyTownPlayer | null | undefined,
): boolean {
  if (gridX < 0 || gridY < 0 || gridX >= map.width || gridY >= map.height || stoneBlockQuantity < 1) {
    return false
  }
  const tileRect = {
    x: gridX * map.tileSize,
    y: gridY * map.tileSize,
    width: map.tileSize,
    height: map.tileSize,
  }
  return !(
    map.blockedRects.some((blocked) => rectsOverlap(tileRect, blocked)) ||
    map.portals.some((portal) => rectsOverlap(tileRect, portal)) ||
    map.npcs.some((npc) => rectsOverlap(tileRect, {
      x: npc.x - playerSize / 2,
      y: npc.y - playerSize / 2,
      width: playerSize,
      height: playerSize,
    })) ||
    worldObjects.some((object) => object.reservesPlacement && rectsOverlap(tileRect, worldObjectRect(object))) ||
    (self && rectsOverlap(tileRect, {
      x: self.x - playerSize / 2,
      y: self.y - playerSize / 2,
      width: playerSize,
      height: playerSize,
    }))
  )
}

export function worldObjectRect(object: SunnyTownWorldObject): Rect {
  if (object.radius && object.radius > 0) {
    return {
      x: object.x - object.radius,
      y: object.y - object.radius,
      width: object.radius * 2,
      height: object.radius * 2,
    }
  }
  return {
    x: object.x,
    y: object.y,
    width: object.width || 0,
    height: object.height || 0,
  }
}

export function movementDirectionForEvent(event: KeyboardEvent): MovementDirection | null {
  switch (event.code) {
    case 'ArrowUp':
    case 'KeyW':
      return 'up'
    case 'ArrowDown':
    case 'KeyS':
      return 'down'
    case 'ArrowLeft':
    case 'KeyA':
      return 'left'
    case 'ArrowRight':
    case 'KeyD':
      return 'right'
  }

  switch (event.key.toLowerCase()) {
    case 'arrowup':
    case 'w':
      return 'up'
    case 'arrowdown':
    case 's':
      return 'down'
    case 'arrowleft':
    case 'a':
      return 'left'
    case 'arrowright':
    case 'd':
      return 'right'
    default:
      return null
  }
}

function movementVector(input: MovementInput): { x: number; y: number } | null {
  let x = Number(input.right) - Number(input.left)
  let y = Number(input.down) - Number(input.up)
  if (x === 0 && y === 0) {
    return null
  }

  const length = Math.hypot(x, y)
  x /= length
  y /= length
  return { x, y }
}

function movementFacing(vector: { x: number; y: number }): SunnyTownPlayer['facing'] {
  if (Math.abs(vector.x) > Math.abs(vector.y)) {
    return vector.x > 0 ? 'right' : 'left'
  }
  return vector.y > 0 ? 'down' : 'up'
}

function collides(map: SunnyTownMap, worldObjects: SunnyTownWorldObject[], x: number, y: number): boolean {
  const playerRect = {
    x: x - playerSize / 2,
    y: y - playerSize / 2,
    width: playerSize,
    height: playerSize,
  }
  return (
    map.blockedRects.some((blocked) => rectsOverlap(playerRect, blocked)) ||
    worldObjects.some((object) => object.active && object.collision && rectsOverlap(playerRect, worldObjectRect(object)))
  )
}

function clampPlayerX(map: SunnyTownMap, x: number): number {
  const maxX = map.width * map.tileSize - playerSize / 2
  return clamp(x, playerSize / 2, maxX)
}

function clampPlayerY(map: SunnyTownMap, y: number): number {
  const maxY = map.height * map.tileSize - playerSize / 2
  return clamp(y, playerSize / 2, maxY)
}

function rectsOverlap(first: Rect, second: Rect): boolean {
  return (
    first.x < second.x + second.width &&
    first.x + first.width > second.x &&
    first.y < second.y + second.height &&
    first.y + first.height > second.y
  )
}

function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value))
}
