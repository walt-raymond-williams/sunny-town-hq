import { ref } from 'vue'
import type { SunnyTownMap, SunnyTownPlayer, SunnyTownWorldObject } from '../types/sunnyTown'
import { canPlaceStoneBlock as canPlaceStoneBlockOnMap } from './useSunnyTownMovement'

export interface PlacementGrid {
  gridX: number
  gridY: number
}

export interface CameraPosition {
  x: number
  y: number
}

export interface PointerRect {
  left: number
  top: number
  width: number
  height: number
}

export function useSunnyTownPlacement() {
  const hoverGrid = ref<PlacementGrid | null>(null)

  function clearHover() {
    hoverGrid.value = null
  }

  function setHoverFromPointer(
    event: PointerEvent,
    map: SunnyTownMap | null,
    canvas: HTMLCanvasElement | null,
    camera: CameraPosition,
  ): PlacementGrid | null {
    hoverGrid.value = gridFromPointer(event, map, canvas, camera)
    return hoverGrid.value
  }

  function gridFromPointer(
    event: PointerEvent,
    map: SunnyTownMap | null,
    canvas: HTMLCanvasElement | null,
    camera: CameraPosition,
  ): PlacementGrid | null {
    if (!map || !canvas) {
      return null
    }
    return gridFromPointerPosition(map, canvas.getBoundingClientRect(), camera, event.clientX, event.clientY)
  }

  function canPlaceStoneBlock(
    map: SunnyTownMap,
    gridX: number,
    gridY: number,
    stoneBlockQuantity: number,
    worldObjects: SunnyTownWorldObject[],
    self: SunnyTownPlayer | null | undefined,
  ): boolean {
    return canPlaceStoneBlockOnMap(map, gridX, gridY, stoneBlockQuantity, worldObjects, self)
  }

  return {
    canPlaceStoneBlock,
    clearHover,
    gridFromPointer,
    hoverGrid,
    setHoverFromPointer,
  }
}

export function gridFromPointerPosition(
  map: SunnyTownMap,
  rect: PointerRect,
  camera: CameraPosition,
  clientX: number,
  clientY: number,
): PlacementGrid | null {
  const worldX = clientX - rect.left + camera.x
  const worldY = clientY - rect.top + camera.y
  const gridX = Math.floor(worldX / map.tileSize)
  const gridY = Math.floor(worldY / map.tileSize)
  if (gridX < 0 || gridY < 0 || gridX >= map.width || gridY >= map.height) {
    return null
  }
  return { gridX, gridY }
}
