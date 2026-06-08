import type { StudentHotbar, StudentInventory } from './inventory'

export interface SunnyTownSession {
  roomId: string
  mapId: string
  characterId: number
  avatarId: string
  websocketUrl: string
  joinToken: string
  expiresAt: string
  wallet: SunnyTownWallet
  inventory: StudentInventory
  hotbar: StudentHotbar
}

export interface SunnyTownWallet {
  starBalance: number
}

export interface SunnyTownPlayer {
  id: string
  characterId?: number
  displayName: string
  x: number
  y: number
  facing: 'up' | 'down' | 'left' | 'right'
  moving: boolean
  avatarId: string
  equipment?: SunnyTownEquipment
  lastProcessedSeq: number
}

export interface SunnyTownEquipment {
  gear?: string
  accessory?: string
  tool?: string
}

export interface SunnyTownPortal {
  id: string
  x: number
  y: number
  width: number
  height: number
  targetMapId: string
  targetX: number
  targetY: number
  targetFacing: SunnyTownPlayer['facing']
}

export interface SunnyTownNpc {
  id: string
  characterId?: number
  name: string
  x: number
  y: number
  facing: SunnyTownPlayer['facing']
  moving?: boolean
  spriteKey: string
  dialogue: string[]
  shop?: SunnyTownShop
  activity?: SunnyTownActivity
}

export interface SunnyTownLocation {
  id: string
  name: string
  x: number
  y: number
  radius: number
  tags: string[]
  ownerNpcKey?: string
  capacity?: number
}

export interface SunnyTownActivity {
  type: 'schoolwork'
}

export interface SunnyTownShop {
  id: string
  items: SunnyTownShopItem[]
}

export interface SunnyTownShopItem {
  itemKey: string
  name: string
  description: string
  priceStars: number
}

export interface SunnyTownMap {
  id: string
  name: string
  tileSize: number
  width: number
  height: number
  spawns: Array<{ x: number; y: number }>
  blockedRects: Array<{ x: number; y: number; width: number; height: number }>
  starSpawns: Array<{ x: number; y: number }>
  portals: SunnyTownPortal[]
  npcs: SunnyTownNpc[]
  resourceNodes: SunnyTownResourceNodeDefinition[]
  locations?: SunnyTownLocation[]
}

export interface SunnyTownPlacedObject {
  id: string
  itemKey: 'stone_block'
  gridX: number
  gridY: number
  x: number
  y: number
  width: number
  height: number
  placedByAppUserId?: number
}

export interface SunnyTownCollectible {
  id: string
  kind: 'star'
  x: number
  y: number
  active: boolean
}

export interface SunnyTownResourceNodeDefinition {
  id: string
  kind: 'rock'
  x: number
  y: number
  radius: number
  interactionRadius: number
  respawnSeconds: number
}

export interface SunnyTownResourceNode {
  id: string
  kind: 'rock'
  x: number
  y: number
  radius: number
  active: boolean
  hits: number
  needed: number
}

export interface SunnyTownWorldObject {
  id: string
  kind: 'rock_node' | 'stone_block'
  source: 'natural' | 'placed'
  itemKey?: 'stone_block'
  resourceKind?: 'rock'
  x: number
  y: number
  width?: number
  height?: number
  radius?: number
  active: boolean
  collision: boolean
  breakable: boolean
  reservesPlacement: boolean
  hits?: number
  needed?: number
  gridX?: number
  gridY?: number
  placedByAppUserId?: number
}

export interface SunnyTownServerMessage {
  type: 'hello' | 'snapshot' | 'map_changed' | 'error' | 'reward_committed' | 'reward_failed' | 'resource_committed' | 'resource_failed' | 'map_object_placed' | 'map_object_removed'
  selfId?: string
  roomId?: string
  mapId?: string
  map?: SunnyTownMap
  tick?: number
  serverTimeMs?: number
  players?: SunnyTownPlayer[]
  npcs?: SunnyTownNpc[]
  collectibles?: SunnyTownCollectible[]
  resourceNodes?: SunnyTownResourceNode[]
  placedObjects?: SunnyTownPlacedObject[]
  placedObject?: SunnyTownPlacedObject
  worldObjects?: SunnyTownWorldObject[]
  worldObject?: SunnyTownWorldObject
  code?: string
  eventId?: string
  kind?: 'star'
  amount?: number
  newStarBalance?: number
  collectibleId?: string
  nodeId?: string
  resourceKey?: 'rock' | 'crystal' | 'stone_block'
  quantity?: number
  reason?: string
}

export interface SunnyTownMoveMessage {
  type: 'move'
  seq: number
  x: number
  y: number
  facing: 'up' | 'down' | 'left' | 'right'
  moving: boolean
  clientTimeMs: number
}

export interface SunnyTownEquipmentChangedMessage {
  type: 'equipment_changed'
}

export interface SunnyTownToolUseMessage {
  type: 'tool_use'
  toolKey: string
  x: number
  y: number
  facing: 'up' | 'down' | 'left' | 'right'
  clientTimeMs: number
}

export interface SunnyTownPlaceObjectMessage {
  type: 'place_object'
  itemKey: 'stone_block'
  gridX: number
  gridY: number
  clientTimeMs: number
}
