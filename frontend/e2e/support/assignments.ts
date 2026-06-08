import { hqFetch } from './auth'

interface Assignment {
  id: number
  prompt: string
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
