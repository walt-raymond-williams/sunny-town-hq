import { authJson } from './http'
import type { StudentSummary } from '../types/user'

export async function getStudents(): Promise<StudentSummary[]> {
  return authJson<StudentSummary[]>('/api/students')
}
