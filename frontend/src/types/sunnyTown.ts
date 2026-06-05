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
  lastProcessedSeq: number
}

export interface SunnyTownCollectible {
  id: string
  kind: 'star'
  x: number
  y: number
  active: boolean
}

export interface SunnyTownServerMessage {
  type: 'hello' | 'snapshot' | 'error' | 'reward_committed' | 'reward_failed'
  selfId?: string
  roomId?: string
  mapId?: string
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

export interface SunnyTownInputMessage {
  type: 'input'
  seq: number
  up: boolean
  down: boolean
  left: boolean
  right: boolean
}
