import { authJson } from './http'
import type { SunnyTownSession } from '../types/sunnyTown'
import type { StudentHotbar } from '../types/inventory'
import { normalizeHotbar } from './hotbarApi'
import { normalizeInventoryItem, type StudentInventoryResponse } from './inventoryApi'

interface SunnyTownSessionResponse {
  room_id?: string
  map_id?: string
  character_id?: number
  avatar_id?: string
  websocket_url?: string
  join_token?: string
  expires_at?: string
  roomId?: string
  mapId?: string
  characterId?: number
  websocketUrl?: string
  joinToken?: string
  expiresAt?: string
  avatar?: {
    id?: string
  }
  wallet?: {
    star_balance: number
  }
  inventory?: StudentInventoryResponse
  hotbar?: StudentHotbar
}

export async function createSunnyTownSession(): Promise<SunnyTownSession> {
  const response = await authJson<SunnyTownSessionResponse>('/api/student/sunny-town/session', {
    method: 'POST',
  })
  return {
    roomId: response.room_id || response.roomId || 'sunny-town-main',
    mapId: response.map_id || response.mapId || 'sunny-town-v1',
    characterId: response.character_id || response.characterId || 0,
    avatarId: response.avatar_id || response.avatar?.id || 'pet-default',
    websocketUrl: response.websocket_url || response.websocketUrl || 'ws://127.0.0.1:18082/sunny-town/ws',
    joinToken: response.join_token || response.joinToken || '',
    expiresAt: response.expires_at || response.expiresAt || '',
    wallet: {
      starBalance: response.wallet?.star_balance ?? 0,
    },
    inventory: {
      items: (response.inventory?.items || []).map(normalizeInventoryItem).filter((item) => item.quantity > 0),
    },
    hotbar: normalizeHotbar(response.hotbar),
  }
}
