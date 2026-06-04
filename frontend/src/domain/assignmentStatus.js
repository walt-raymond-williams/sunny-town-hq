export function currentAttempt(assignment) {
  return assignment?.current_attempt || null
}

export function attemptHistory(assignment) {
  return assignment?.attempts || []
}

export function previousAttempts(assignment) {
  return attemptHistory(assignment)
}

export function gradedAttempts(assignment) {
  return attemptHistory(assignment).filter((attempt) => attempt.passed !== null)
}

export function latestGradedAttempt(assignment) {
  const attempts = gradedAttempts(assignment)
  return attempts.length > 0 ? attempts[attempts.length - 1] : null
}

export function hasPreviousAttempts(assignment) {
  return previousAttempts(assignment).length > 0
}

export function statusLabel(assignment) {
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

export function statusColor(assignment) {
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

export function categoryPercentLabel(summary) {
  return summary.percent === null ? 'No graded work' : `${summary.percent}% passed`
}

export function passFailLabel(passed) {
  return passed ? 'Passed' : 'Failed'
}

export function passFailColor(passed) {
  return passed ? 'success' : 'error'
}

export function attemptStatusLabel(attempt) {
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

export function attemptStatusColor(attempt) {
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

export function answerPreview(answer) {
  if (!answer) {
    return ''
  }

  return answer.length > 90 ? `${answer.slice(0, 90)}...` : answer
}
