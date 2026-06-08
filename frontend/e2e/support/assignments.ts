import { hqFetch } from './auth'

export interface Attempt {
  id: number
  submitted_answer: string
  passed: boolean | null
  feedback: string | null
  reset_at: string | null
}

export interface Assignment {
  id: number
  category: string
  prompt: string
  expected_answer: string
  current_attempt?: Attempt | null
  attempts: Attempt[]
}

export async function resetAssignments(baseURL: string) {
  if (process.env.PLAYWRIGHT_PRESERVE_ASSIGNMENTS === 'true') {
    return
  }

  const assignments = await hqFetch<Assignment[]>('teacher', baseURL, '/api/assignments')
  await Promise.all(
    assignments.map((assignment) =>
      hqFetch<void>('teacher', baseURL, `/api/assignments/${assignment.id}`, { method: 'DELETE' }),
    ),
  )
}

export async function createAssignment(
  baseURL: string,
  payload: { category?: string; prompt: string; expectedAnswer: string },
): Promise<Assignment> {
  return hqFetch<Assignment>('teacher', baseURL, '/api/assignments', {
    method: 'POST',
    body: JSON.stringify({
      category: payload.category || 'MATH',
      prompt: payload.prompt,
      expected_answer: payload.expectedAnswer,
    }),
  })
}

export async function submitStudentAssignment(baseURL: string, assignmentID: number, answer: string): Promise<void> {
  await hqFetch<void>('student', baseURL, `/api/assignments/${assignmentID}/submit`, {
    method: 'POST',
    body: JSON.stringify({ submitted_answer: answer }),
  })
}

export async function gradeAssignment(
  baseURL: string,
  assignmentID: number,
  payload: { attemptID: number; passed: boolean; feedback: string },
): Promise<void> {
  await hqFetch<void>('teacher', baseURL, `/api/assignments/${assignmentID}/grade`, {
    method: 'PATCH',
    body: JSON.stringify({
      attempt_id: payload.attemptID,
      passed: payload.passed,
      feedback: payload.feedback,
    }),
  })
}

export async function awardCookieThroughSchoolwork(baseURL: string, runID: string): Promise<void> {
  const assignment = await createAssignment(baseURL, {
    prompt: `Playwright ${runID} cookie award`,
    expectedAnswer: `Cookie answer ${runID}`,
  })
  await submitStudentAssignment(baseURL, assignment.id, `Cookie response ${runID}`)
  const assignments = await hqFetch<Assignment[]>('teacher', baseURL, '/api/assignments')
  const current = assignments.find((item) => item.id === assignment.id)?.current_attempt
  if (!current) {
    throw new Error('Cookie award setup did not create a current attempt')
  }
  await gradeAssignment(baseURL, assignment.id, {
    attemptID: current.id,
    passed: true,
    feedback: `Cookie feedback ${runID}`,
  })
}
