import { authFetch } from '../auth'
import { authJson, jsonOptions, readJson } from './http'
import type {
  Assignment,
  CreateAssignmentPayload,
  GradeAssignmentPayload,
  ResetAssignmentPayload,
  TeacherStudentFilter,
} from '../types/assignment'

export async function getAssignments(studentID: TeacherStudentFilter = 'ALL'): Promise<Assignment[]> {
  const query = new URLSearchParams()
  if (studentID !== 'ALL') {
    query.set('student_id', studentID)
  }

  const endpoint = query.toString() ? `/api/assignments?${query.toString()}` : '/api/assignments'
  return authJson<Assignment[]>(endpoint)
}

export async function createAssignment(payload: CreateAssignmentPayload): Promise<Assignment> {
  return authJson<Assignment>('/api/assignments', jsonOptions('POST', payload))
}

export async function deleteAssignment(id: number): Promise<void> {
  const response = await authFetch(`/api/assignments/${id}`, {
    method: 'DELETE',
  })

  if (!response.ok) {
    let message = `API returned ${response.status}`
    try {
      const body = await response.json()
      message = body.error || message
    } catch {
      // Keep the status-code fallback.
    }
    throw new Error(message)
  }
}

export async function gradeAssignment(
  id: number,
  payload: GradeAssignmentPayload,
): Promise<Assignment> {
  return readJson<Assignment>(
    await authFetch(`/api/assignments/${id}/grade`, jsonOptions('PATCH', payload)),
  )
}

export async function resetAssignment(
  id: number,
  payload: ResetAssignmentPayload,
): Promise<Assignment> {
  return readJson<Assignment>(
    await authFetch(`/api/assignments/${id}/reset`, jsonOptions('PATCH', payload)),
  )
}
