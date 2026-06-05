import type { Assignment, AssignmentAttempt, GradeSummary } from '../types/assignment'

export function currentAttempt(assignment: Assignment | null | undefined): AssignmentAttempt | null {
  return assignment?.current_attempt || null
}

export function attemptHistory(assignment: Assignment | null | undefined): AssignmentAttempt[] {
  return assignment?.attempts || []
}

export function previousAttempts(assignment: Assignment | null | undefined): AssignmentAttempt[] {
  return attemptHistory(assignment)
}

export function gradedAttempts(assignment: Assignment | null | undefined): AssignmentAttempt[] {
  return attemptHistory(assignment).filter((attempt) => attempt.passed !== null)
}

export function latestGradedAttempt(
  assignment: Assignment | null | undefined,
): AssignmentAttempt | null {
  const attempts = gradedAttempts(assignment)
  return attempts.length > 0 ? (attempts[attempts.length - 1] ?? null) : null
}

export function hasPreviousAttempts(assignment: Assignment | null | undefined): boolean {
  return previousAttempts(assignment).length > 0
}

export function statusLabel(assignment: Assignment): string {
  const attempt = currentAttempt(assignment)
  if (!attempt) {
    return attemptHistory(assignment).length > 0 ? 'Reset' : 'Unanswered'
  }

  if (attempt.passed === true) {
    return 'Passed'
  }

  if (attempt.passed === false) {
    return 'Failed'
  }

  return 'Needs Review'
}

export function statusColor(assignment: Assignment): string {
  const attempt = currentAttempt(assignment)
  if (!attempt) {
    return attemptHistory(assignment).length > 0 ? 'warning' : 'info'
  }

  if (attempt.passed === true) {
    return 'success'
  }

  if (attempt.passed === false) {
    return 'error'
  }

  return 'warning'
}

export function categoryPercentLabel(summary: GradeSummary): string {
  return summary.percent === null ? 'No graded work' : `${summary.percent}% passed`
}

export function passFailLabel(passed: boolean | null | undefined): string {
  return passed ? 'Passed' : 'Failed'
}

export function passFailColor(passed: boolean | null | undefined): string {
  return passed ? 'success' : 'error'
}

export function attemptStatusLabel(attempt: AssignmentAttempt): string {
  if (attempt.reset_at) {
    return 'Reset'
  }

  if (attempt.passed === true) {
    return 'Passed'
  }

  if (attempt.passed === false) {
    return 'Failed'
  }

  return 'Needs Review'
}

export function attemptStatusColor(attempt: AssignmentAttempt): string {
  if (attempt.reset_at) {
    return 'warning'
  }

  if (attempt.passed === true) {
    return 'success'
  }

  if (attempt.passed === false) {
    return 'error'
  }

  return 'warning'
}

export function answerPreview(answer: string | null | undefined): string {
  if (!answer) {
    return ''
  }

  return answer.length > 90 ? `${answer.slice(0, 90)}...` : answer
}
