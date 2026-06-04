import { authJson } from './http'

export async function getStudents() {
  return authJson('/api/students')
}
