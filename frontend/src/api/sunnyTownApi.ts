import { authJson } from './http'
import type { SunnyTownSession } from '../types/sunnyTown'

export async function createSunnyTownSession(): Promise<SunnyTownSession> {
  return authJson<SunnyTownSession>('/api/student/sunny-town/session', {
    method: 'POST',
  })
}
