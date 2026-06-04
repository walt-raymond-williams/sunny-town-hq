import { authFetch } from '../auth'
import { authJson, jsonOptions, readJson } from './http'

export async function getAssignments(studentID = 'ALL') {
  const query = new URLSearchParams()
  if (studentID !== 'ALL') {
    query.set('student_id', studentID)
  }

  const endpoint = query.toString() ? `/api/assignments?${query.toString()}` : '/api/assignments'
  return authJson(endpoint)
}

export async function createAssignment(payload) {
  return authJson('/api/assignments', jsonOptions('POST', payload))
}

export async function deleteAssignment(id) {
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

export async function gradeAssignment(id, payload) {
  return readJson(await authFetch(`/api/assignments/${id}/grade`, jsonOptions('PATCH', payload)))
}

export async function resetAssignment(id, payload) {
  return readJson(await authFetch(`/api/assignments/${id}/reset`, jsonOptions('PATCH', payload)))
}
