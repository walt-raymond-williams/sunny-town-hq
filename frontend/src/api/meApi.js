import { authJson } from './http'

export async function getMe() {
  return authJson('/api/me')
}
