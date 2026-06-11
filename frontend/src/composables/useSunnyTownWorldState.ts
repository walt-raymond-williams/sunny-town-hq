import { ref } from 'vue'
import type {
  SunnyTownCollectible,
  SunnyTownMap,
  SunnyTownNpc,
  SunnyTownPlacedObject,
  SunnyTownPlayer,
  SunnyTownResourceNode,
  SunnyTownServerMessage,
  SunnyTownWorldObject,
} from '../types/sunnyTown'
import {
  legacyWorldObjects,
  placedObjectToWorldObject,
  sameWorldObject,
} from '../features/sunny-town/worldObjects'

export function normalizeSunnyTownMap(map: SunnyTownMap): SunnyTownMap {
  return {
    ...map,
    portals: map.portals || [],
    npcs: map.npcs || [],
    resourceNodes: map.resourceNodes || [],
    fixtures: map.fixtures || [],
    blockedRects: map.blockedRects || [],
    starSpawns: map.starSpawns || [],
    spawns: map.spawns || [],
  }
}

export function useSunnyTownWorldState() {
  const activeMap = ref<SunnyTownMap | null>(null)
  const players = ref<SunnyTownPlayer[]>([])
  const npcs = ref<SunnyTownNpc[]>([])
  const collectibles = ref<SunnyTownCollectible[]>([])
  const resourceNodes = ref<SunnyTownResourceNode[]>([])
  const placedObjects = ref<SunnyTownPlacedObject[]>([])
  const worldObjects = ref<SunnyTownWorldObject[]>([])

  function applySnapshot(message: SunnyTownServerMessage): boolean {
    if (message.mapId && activeMap.value && message.mapId !== activeMap.value.id) {
      return false
    }
    players.value = message.players || []
    npcs.value = message.npcs !== undefined ? message.npcs : npcs.value
    collectibles.value = message.collectibles || []
    resourceNodes.value = message.resourceNodes || []
    placedObjects.value = message.placedObjects || placedObjects.value
    worldObjects.value = message.worldObjects || legacyWorldObjects(resourceNodes.value, placedObjects.value)
    return true
  }

  function applyMapState(message: SunnyTownServerMessage) {
    if (message.map) {
      activeMap.value = normalizeSunnyTownMap(message.map)
    }
    players.value = message.players || []
    npcs.value = message.npcs !== undefined ? message.npcs : activeMap.value?.npcs || []
    collectibles.value = message.collectibles || []
    resourceNodes.value = message.resourceNodes || []
    placedObjects.value = message.placedObjects || []
    worldObjects.value = message.worldObjects || legacyWorldObjects(resourceNodes.value, placedObjects.value)
  }

  function applyPlacedObject(message: SunnyTownServerMessage) {
    if (message.placedObject) {
      placedObjects.value = [
        ...placedObjects.value.filter((object) => object.id !== message.placedObject?.id),
        message.placedObject,
      ]
    }
    const placedWorldObject = message.worldObject
    if (placedWorldObject) {
      worldObjects.value = [
        ...worldObjects.value.filter((object) => !sameWorldObject(object, placedWorldObject)),
        placedWorldObject,
      ]
    } else if (message.placedObject) {
      worldObjects.value = [
        ...worldObjects.value.filter((object) => object.source !== 'placed' || object.id !== message.placedObject?.id),
        placedObjectToWorldObject(message.placedObject),
      ]
    }
  }

  function applyRemovedObject(message: SunnyTownServerMessage) {
    if (message.placedObject) {
      placedObjects.value = placedObjects.value.filter((object) => object.id !== message.placedObject?.id)
    }
    const removedWorldObject = message.worldObject
    if (removedWorldObject) {
      worldObjects.value = worldObjects.value.filter((object) => !sameWorldObject(object, removedWorldObject))
    } else if (message.placedObject) {
      worldObjects.value = worldObjects.value.filter((object) => object.source !== 'placed' || object.id !== message.placedObject?.id)
    }
  }

  return {
    activeMap,
    applyMapState,
    applyPlacedObject,
    applyRemovedObject,
    applySnapshot,
    collectibles,
    npcs,
    placedObjects,
    players,
    resourceNodes,
    worldObjects,
  }
}
