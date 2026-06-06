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
  name: string
  x: number
  y: number
  facing: SunnyTownPlayer['facing']
  spriteKey: string
  dialogue: string[]
  shop?: SunnyTownShop
  activity?: SunnyTownActivity
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

export interface SunnyTownServerMessage {
  type: 'hello' | 'snapshot' | 'map_changed' | 'error' | 'reward_committed' | 'reward_failed' | 'resource_committed' | 'resource_failed'
  selfId?: string
  roomId?: string
  mapId?: string
  map?: SunnyTownMap
  tick?: number
  serverTimeMs?: number
  players?: SunnyTownPlayer[]
  collectibles?: SunnyTownCollectible[]
  resourceNodes?: SunnyTownResourceNode[]
  code?: string
  eventId?: string
  kind?: 'star'
  amount?: number
  newStarBalance?: number
  collectibleId?: string
  nodeId?: string
  resourceKey?: 'rock' | 'crystal'
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
