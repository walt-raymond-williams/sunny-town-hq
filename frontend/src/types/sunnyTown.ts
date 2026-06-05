export interface SunnyTownAvatar {
  id: string
  displayName: string
}

export interface SunnyTownSession {
  roomId: string
  mapId: string
  websocketUrl: string
  joinToken: string
  expiresAt: string
  avatar: SunnyTownAvatar
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

export interface SunnyTownServerMessage {
  type: 'hello' | 'snapshot' | 'error'
  selfId?: string
  roomId?: string
  mapId?: string
  tick?: number
  serverTimeMs?: number
  players?: SunnyTownPlayer[]
  code?: string
}

export interface SunnyTownInputMessage {
  type: 'input'
  seq: number
  up: boolean
  down: boolean
  left: boolean
  right: boolean
}
