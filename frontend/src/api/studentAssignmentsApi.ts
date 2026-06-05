import { authJson, jsonOptions } from './http'
import type {
  Assignment,
  NextStudentAssignmentResponse,
  StudentCategoryFilter,
  SubmitAssignmentPayload,
} from '../types/assignment'

export async function getNextStudentAssignment(
  category: StudentCategoryFilter = 'ALL',
): Promise<Assignment | null> {
  const query = new URLSearchParams()
  if (category !== 'ALL') {
    query.set('category', category)
  }

  const endpoint = query.toString()
    ? `/api/student/assignments/next?${query.toString()}`
    : '/api/student/assignments/next'
  const body = await authJson<NextStudentAssignmentResponse>(endpoint)
  return body.assignment
}

export async function getStudentGradedAssignments(): Promise<Assignment[]> {
  return authJson<Assignment[]>('/api/student/assignments/graded')
}

export async function submitStudentAnswer(id: number, answer: string): Promise<Assignment> {
  return authJson<Assignment>(
    `/api/assignments/${id}/submit`,
    jsonOptions<SubmitAssignmentPayload>('POST', {
      submitted_answer: answer,
    }),
  )
}
