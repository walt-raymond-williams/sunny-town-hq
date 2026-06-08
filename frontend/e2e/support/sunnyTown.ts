import { hqFetch } from './auth'

interface AuthUser {
  id: number
}

const serviceSecret = process.env.PLAYWRIGHT_HQ_SERVICE_SECRET || 'local-dev-service-secret'

export async function currentStudentID(baseURL: string): Promise<number> {
  const me = await hqFetch<AuthUser>('student', baseURL, '/api/me')
  return me.id
}

export async function setSunnyTownPosition(
  baseURL: string,
  position: { mapId: string; x: number; y: number; facing?: 'up' | 'down' | 'left' | 'right' },
): Promise<void> {
  const appUserID = await currentStudentID(baseURL)
  await internalHQFetch(baseURL, '/api/internal/sunny-town/player-position', {
    method: 'POST',
    body: JSON.stringify({
      app_user_id: appUserID,
      room_id: 'sunny-town-main',
      map_id: position.mapId,
      x: position.x,
      y: position.y,
      facing: position.facing || 'up',
    }),
  })
}

export async function seedStudentStars(baseURL: string, amount: number, runID: string): Promise<void> {
  const appUserID = await currentStudentID(baseURL)
  for (let index = 0; index < amount; index += 1) {
    await internalHQFetch(baseURL, '/api/internal/sunny-town/reward-events', {
      method: 'POST',
      body: JSON.stringify({
        event_id: `playwright-${runID}-star-${index}`,
        app_user_id: appUserID,
        room_id: 'sunny-town-main',
        map_id: 'sunny-town-v1',
        collectible_id: `playwright-star-${runID}-${index}`,
        reward_kind: 'star',
        amount: 1,
      }),
    })
  }
}

async function internalHQFetch(baseURL: string, path: string, init: RequestInit): Promise<void> {
  const response = await fetch(new URL(path, baseURL), {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      'X-HQ-Service-Secret': serviceSecret,
      ...init.headers,
    },
  })
  if (!response.ok) {
    throw new Error(`${init.method || 'GET'} ${path} failed: ${response.status} ${await response.text()}`)
  }
}
