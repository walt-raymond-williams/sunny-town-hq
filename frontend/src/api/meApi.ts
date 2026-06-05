import { authJson } from './http'
import type { AuthUser } from '../types/user'

export async function getMe(): Promise<AuthUser> {
  return authJson<AuthUser>('/api/me')
}
