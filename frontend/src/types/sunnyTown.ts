export interface SunnyTownSession {
  roomId: string
  mapId: string
  avatarId: string
  websocketUrl: string
  joinToken: string
  expiresAt: string
  wallet: SunnyTownWallet
}

export interface SunnyTownWallet {
  starBalance: number
}

export interface SunnyTownPlayer {
  id: string
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
  name: string
  x: number
  y: number
  facing: SunnyTownPlayer['facing']
  spriteKey: string
  dialogue: string[]
  shop?: SunnyTownShop
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
}

export interface SunnyTownCollectible {
  id: string
  kind: 'star'
  x: number
  y: number
  active: boolean
}

export interface SunnyTownServerMessage {
  type: 'hello' | 'snapshot' | 'map_changed' | 'error' | 'reward_committed' | 'reward_failed'
  selfId?: string
  roomId?: string
  mapId?: string
  map?: SunnyTownMap
  tick?: number
  serverTimeMs?: number
  players?: SunnyTownPlayer[]
  collectibles?: SunnyTownCollectible[]
  code?: string
  eventId?: string
  kind?: 'star'
  amount?: number
  newStarBalance?: number
  collectibleId?: string
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
