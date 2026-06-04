import { authJson, jsonOptions } from './http'

export async function getNextStudentAssignment(category = 'ALL') {
  const query = new URLSearchParams()
  if (category !== 'ALL') {
    query.set('category', category)
  }

  const endpoint = query.toString()
    ? `/api/student/assignments/next?${query.toString()}`
    : '/api/student/assignments/next'
  const body = await authJson(endpoint)
  return body.assignment
}

export async function getStudentGradedAssignments() {
  return authJson('/api/student/assignments/graded')
}

export async function submitStudentAnswer(id, answer) {
  return authJson(
    `/api/assignments/${id}/submit`,
    jsonOptions('POST', {
      submitted_answer: answer,
    }),
  )
}
