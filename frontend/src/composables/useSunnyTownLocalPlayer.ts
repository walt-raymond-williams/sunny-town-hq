import type { MovementInput } from './useSunnyTownMovement'
import { simulatePlayer } from './useSunnyTownMovement'
import type {
  SunnyTownEquipment,
  SunnyTownMap,
  SunnyTownPlayer,
  SunnyTownWorldObject,
} from '../types/sunnyTown'

export function useSunnyTownLocalPlayer() {
  let localSelf: SunnyTownPlayer | null = null
  let renderedSelf: SunnyTownPlayer | null = null

  function clear() {
    localSelf = null
    renderedSelf = null
  }

  function current(players: SunnyTownPlayer[], selfId: string): SunnyTownPlayer | null {
    return localSelf || players.find((player) => player.id === selfId) || null
  }

  function local(): SunnyTownPlayer | null {
    return localSelf
  }

  function syncFromSnapshot(selfSnapshot: SunnyTownPlayer | null | undefined) {
    if (!selfSnapshot) {
      clear()
      return
    }

    if (!localSelf) {
      localSelf = { ...selfSnapshot }
      return
    }

    localSelf.displayName = selfSnapshot.displayName
    localSelf.avatarId = selfSnapshot.avatarId
    localSelf.equipment = selfSnapshot.equipment ? { ...selfSnapshot.equipment } : undefined
    localSelf.lastProcessedSeq = selfSnapshot.lastProcessedSeq
  }

  function refreshMovement(
    input: MovementInput,
    map: SunnyTownMap | null,
    worldObjects: SunnyTownWorldObject[],
  ): SunnyTownPlayer | null {
    if (!localSelf) {
      return null
    }
    localSelf = simulatePlayer(localSelf, input, map, worldObjects, 0)
    return localSelf
  }

  function predict(
    selfSnapshot: SunnyTownPlayer,
    input: MovementInput,
    map: SunnyTownMap,
    worldObjects: SunnyTownWorldObject[],
    deltaSeconds: number,
  ): SunnyTownPlayer {
    if (!localSelf) {
      localSelf = { ...selfSnapshot }
    }

    localSelf = simulatePlayer(localSelf, input, map, worldObjects, deltaSeconds)
    return localSelf
  }

  function renderedPlayers(
    selfSnapshot: SunnyTownPlayer | null | undefined,
    renderedRemotePlayers: SunnyTownPlayer[],
  ): SunnyTownPlayer[] {
    if (!selfSnapshot) {
      return renderedRemotePlayers
    }

    if (!localSelf) {
      localSelf = { ...selfSnapshot }
    }
    if (!renderedSelf) {
      renderedSelf = { ...localSelf }
    }

    copyRenderedSelf(renderedSelf, localSelf)
    return [...renderedRemotePlayers, renderedSelf]
  }

  function applyEquipment(equipment: SunnyTownEquipment) {
    if (localSelf) {
      localSelf.equipment = { ...equipment }
    }
    if (renderedSelf) {
      renderedSelf.equipment = { ...equipment }
    }
  }

  return {
    applyEquipment,
    clear,
    current,
    local,
    predict,
    refreshMovement,
    renderedPlayers,
    syncFromSnapshot,
  }
}

function copyRenderedSelf(renderedSelf: SunnyTownPlayer, target: SunnyTownPlayer) {
  renderedSelf.x = target.x
  renderedSelf.y = target.y
  renderedSelf.displayName = target.displayName
  renderedSelf.facing = target.facing
  renderedSelf.moving = target.moving
  renderedSelf.avatarId = target.avatarId
  renderedSelf.equipment = target.equipment ? { ...target.equipment } : undefined
  renderedSelf.lastProcessedSeq = target.lastProcessedSeq
}
