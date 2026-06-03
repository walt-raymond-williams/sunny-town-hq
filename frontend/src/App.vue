<script setup>
import { computed, reactive, ref } from 'vue'
import StudentPet from './components/StudentPet.vue'

const categories = ['MATH', 'SCIENCE', 'READING']
const passFailOptions = [
  { label: 'Pass', value: true },
  { label: 'Fail', value: false },
]

const view = ref('splash')
const studentTab = ref('answer')
const teacherFilter = ref('needs-review')
const isCreatingQuestion = ref(false)
const password = ref('')
const loginError = ref('')
const assignmentMessage = ref('')
const assignmentError = ref('')
const gradingMessage = ref('')
const gradingError = ref('')
const assignments = ref([])
const gradingForms = reactive({})
const studentGradedAssignments = ref([])
const studentAssignment = ref(null)
const studentAnswer = ref('')
const studentMessage = ref('')
const studentError = ref('')
const studentGradesError = ref('')
const isLoadingAssignments = ref(false)
const isLoadingStudentAssignment = ref(false)
const isLoadingStudentGrades = ref(false)
const isSaving = ref(false)
const isSavingGrade = ref(false)
const isResettingAssignment = ref(false)
const isSubmittingStudentAnswer = ref(false)
const isLoggingOut = ref(false)

const form = ref({
  category: 'MATH',
  prompt: '',
  expected_answer: '',
})
const teacherFilters = [
  { label: 'Needs Review', value: 'needs-review' },
  { label: 'All', value: 'all' },
  { label: 'Unanswered', value: 'unanswered' },
  { label: 'Reset', value: 'reset' },
  { label: 'Graded', value: 'graded' },
]

const hasAssignments = computed(() => assignments.value.length > 0)
const filteredAssignments = computed(() =>
  assignments.value.filter((assignment) => {
    const attempt = currentAttempt(assignment)
    const hasAttempts = attemptHistory(assignment).length > 0

    if (teacherFilter.value === 'all') {
      return true
    }

    if (teacherFilter.value === 'needs-review') {
      return Boolean(attempt && attempt.passed === null)
    }

    if (teacherFilter.value === 'unanswered') {
      return !attempt && !hasAttempts
    }

    if (teacherFilter.value === 'reset') {
      return !attempt && hasAttempts
    }

    if (teacherFilter.value === 'graded') {
      return Boolean(attempt && attempt.passed !== null)
    }

    return true
  }),
)
const hasFilteredAssignments = computed(() => filteredAssignments.value.length > 0)
const hasStudentGradedAssignments = computed(() =>
  studentGradedAssignments.value.some((assignment) => gradedAttempts(assignment).length > 0),
)
const gradeSummaries = computed(() =>
  categories.map((category) => {
    const attemptsForCategory = studentGradedAssignments.value
      .filter((assignment) => assignment.category === category)
      .flatMap((assignment) => gradedAttempts(assignment))
    const passedCount = attemptsForCategory.filter((attempt) => attempt.passed).length
    const percent =
      attemptsForCategory.length === 0
        ? null
        : Math.round((passedCount / attemptsForCategory.length) * 100)

    return {
      category,
      passedCount,
      total: attemptsForCategory.length,
      percent,
    }
  }),
)
const gradedAssignmentsByCategory = computed(() =>
  categories.map((category) => ({
    category,
    assignments: studentGradedAssignments.value.filter(
      (assignment) => assignment.category === category && gradedAttempts(assignment).length > 0,
    ),
  })),
)

function showSplash() {
  view.value = 'splash'
  loginError.value = ''
}

async function showStudent() {
  view.value = 'student'
  studentTab.value = 'answer'
  await loadNextStudentAssignment()
}

async function handleStudentTabChange(tabName) {
  if (tabName === 'answer') {
    await loadNextStudentAssignment()
  }

  if (tabName === 'grades') {
    await loadStudentGrades()
  }
}

function showTeacherLogin() {
  view.value = 'teacher-login'
  password.value = ''
  loginError.value = ''
}

async function loginTeacher() {
  loginError.value = ''

  try {
    const response = await fetch('/api/teacher/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        password: password.value,
      }),
    })
    const body = await response.json()

    if (!response.ok) {
      throw new Error(body.error || `API returned ${response.status}`)
    }

    password.value = ''
    view.value = 'teacher'
    teacherFilter.value = 'needs-review'
    await loadAssignments()
  } catch (error) {
    loginError.value = error.message
  }
}

async function logoutTeacher() {
  isLoggingOut.value = true
  gradingMessage.value = ''
  gradingError.value = ''
  assignmentMessage.value = ''
  assignmentError.value = ''

  try {
    const response = await fetch('/api/teacher/logout', {
      method: 'POST',
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

    password.value = ''
    teacherFilter.value = 'needs-review'
    assignments.value = []
    view.value = 'splash'
  } catch (error) {
    gradingError.value = error.message
  } finally {
    isLoggingOut.value = false
  }
}

async function loadAssignments() {
  isLoadingAssignments.value = true
  assignmentError.value = ''
  gradingError.value = ''

  try {
    const response = await fetch('/api/assignments')
    const body = await response.json()

    if (!response.ok) {
      throw new Error(body.error || `API returned ${response.status}`)
    }

    assignments.value = body
    assignments.value.forEach((assignment) => {
      syncGradingForm(assignment)
    })
  } catch (error) {
    assignmentError.value = error.message
    gradingError.value = error.message
  } finally {
    isLoadingAssignments.value = false
  }
}

async function saveAssignment() {
  isSaving.value = true
  assignmentMessage.value = ''
  assignmentError.value = ''

  try {
    const payload = {
      category: form.value.category,
      prompt: form.value.prompt.trim(),
      expected_answer: form.value.expected_answer.trim(),
    }

    const response = await fetch('/api/assignments', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    })
    const body = await response.json()

    if (!response.ok) {
      throw new Error(body.error || `API returned ${response.status}`)
    }

    assignmentMessage.value = 'Assignment saved.'
    form.value = {
      category: 'MATH',
      prompt: '',
      expected_answer: '',
    }
    isCreatingQuestion.value = false
    await loadAssignments()
  } catch (error) {
    assignmentError.value = error.message
  } finally {
    isSaving.value = false
  }
}

async function deleteAssignment(assignment) {
  assignmentMessage.value = ''
  assignmentError.value = ''

  try {
    const response = await fetch(`/api/assignments/${assignment.id}`, {
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

    assignments.value = assignments.value.filter((item) => item.id !== assignment.id)
    assignmentMessage.value = 'Assignment deleted.'
    gradingMessage.value = ''
  } catch (error) {
    assignmentError.value = error.message
  }
}

async function saveGrade(assignment) {
  const gradeForm = gradingForms[assignment.id]
  if (!gradeForm) {
    return
  }

  isSavingGrade.value = true
  gradingMessage.value = ''
  gradingError.value = ''

  try {
    const response = await fetch(`/api/assignments/${assignment.id}/grade`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        passed: gradeForm.passed,
        feedback: gradeForm.feedback.trim(),
      }),
    })
    const body = await response.json()

    if (!response.ok) {
      throw new Error(body.error || `API returned ${response.status}`)
    }

    gradingMessage.value = 'Result saved.'
    await loadAssignments()
  } catch (error) {
    gradingError.value = error.message
  } finally {
    isSavingGrade.value = false
  }
}

async function resetAssignment(assignment) {
  const gradeForm = gradingForms[assignment.id]
  isResettingAssignment.value = true
  gradingMessage.value = ''
  gradingError.value = ''

  try {
    const response = await fetch(`/api/assignments/${assignment.id}/reset`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        feedback: gradeForm?.feedback?.trim() || '',
      }),
    })
    const body = await response.json()

    if (!response.ok) {
      throw new Error(body.error || `API returned ${response.status}`)
    }

    gradingMessage.value = 'Assignment reset. Previous attempts were kept.'
    await loadAssignments()
  } catch (error) {
    gradingError.value = error.message
  } finally {
    isResettingAssignment.value = false
  }
}

function syncGradingForm(assignment) {
  const attempt = currentAttempt(assignment)
  gradingForms[assignment.id] = {
    passed: attempt?.passed ?? null,
    feedback: attempt?.feedback || '',
  }
}

function currentAttempt(assignment) {
  return assignment?.current_attempt || null
}

function attemptHistory(assignment) {
  return assignment?.attempts || []
}

function previousAttempts(assignment) {
  return attemptHistory(assignment)
}

function gradedAttempts(assignment) {
  return attemptHistory(assignment).filter((attempt) => attempt.passed !== null)
}

function latestGradedAttempt(assignment) {
  const attempts = gradedAttempts(assignment)
  return attempts.length > 0 ? attempts[attempts.length - 1] : null
}

function hasPreviousAttempts(assignment) {
  return previousAttempts(assignment).length > 0
}

function statusLabel(assignment) {
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

function statusColor(assignment) {
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

function categoryPercentLabel(summary) {
  return summary.percent === null ? 'No graded work' : `${summary.percent}% passed`
}

function passFailLabel(passed) {
  return passed ? 'Passed' : 'Failed'
}

function passFailColor(passed) {
  return passed ? 'success' : 'error'
}

function attemptStatusLabel(attempt) {
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

function attemptStatusColor(attempt) {
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

function answerPreview(answer) {
  if (!answer) {
    return ''
  }

  return answer.length > 90 ? `${answer.slice(0, 90)}...` : answer
}

async function loadNextStudentAssignment() {
  isLoadingStudentAssignment.value = true
  studentAssignment.value = null
  studentAnswer.value = ''
  studentMessage.value = ''
  studentError.value = ''

  try {
    const response = await fetch('/api/student/assignments/next')
    const body = await response.json()

    if (!response.ok) {
      throw new Error(body.error || `API returned ${response.status}`)
    }

    studentAssignment.value = body.assignment
  } catch (error) {
    studentError.value = error.message
  } finally {
    isLoadingStudentAssignment.value = false
  }
}

async function loadStudentGrades() {
  isLoadingStudentGrades.value = true
  studentGradesError.value = ''

  try {
    const response = await fetch('/api/student/assignments/graded')
    const body = await response.json()

    if (!response.ok) {
      throw new Error(body.error || `API returned ${response.status}`)
    }

    studentGradedAssignments.value = body
  } catch (error) {
    studentGradesError.value = error.message
  } finally {
    isLoadingStudentGrades.value = false
  }
}

async function submitStudentAnswer() {
  if (!studentAssignment.value) {
    return
  }

  isSubmittingStudentAnswer.value = true
  studentMessage.value = ''
  studentError.value = ''

  try {
    const response = await fetch(`/api/assignments/${studentAssignment.value.id}/submit`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        submitted_answer: studentAnswer.value.trim(),
      }),
    })
    const body = await response.json()

    if (!response.ok) {
      throw new Error(body.error || `API returned ${response.status}`)
    }

    studentMessage.value = 'Answer submitted.'
    await loadNextStudentAssignment()
  } catch (error) {
    studentError.value = error.message
  } finally {
    isSubmittingStudentAnswer.value = false
  }
}
</script>

<template>
  <v-app>
    <v-main>
      <v-container class="app-container" fluid>
        <v-card v-if="view === 'splash'" class="panel" elevation="8">
          <v-card-text>
            <p class="eyebrow">HQ</p>
            <h1>Headquarters</h1>
            <p class="lead">Choose how you want to use HQ today.</p>
            <div class="actions">
              <v-btn color="primary" variant="tonal" size="large" @click="showStudent">
                Student
              </v-btn>
              <v-btn color="secondary" size="large" @click="showTeacherLogin">
                Teacher
              </v-btn>
            </div>
          </v-card-text>
        </v-card>

        <v-card v-else-if="view === 'teacher-login'" class="panel" elevation="8">
          <v-card-text>
            <v-btn
              class="mb-4"
              color="primary"
              prepend-icon="mdi-arrow-left"
              variant="text"
              @click="showSplash"
            >
              Back
            </v-btn>
            <p class="eyebrow">Teacher</p>
            <h1>Teacher Login</h1>
            <v-form class="form-grid" @submit.prevent="loginTeacher">
              <v-text-field
                v-model="password"
                autocomplete="current-password"
                label="Password"
                type="password"
                variant="outlined"
              />
              <v-btn color="secondary" size="large" type="submit">Enter</v-btn>
            </v-form>
            <v-alert v-if="loginError" class="mt-5" type="error" variant="tonal">
              {{ loginError }}
            </v-alert>
          </v-card-text>
        </v-card>

        <v-card v-else-if="view === 'student'" class="panel student-panel" elevation="8">
          <v-card-text>
            <v-btn
              class="mb-4"
              color="primary"
              prepend-icon="mdi-arrow-left"
              variant="text"
              @click="showSplash"
            >
              Back
            </v-btn>
            <p class="eyebrow">Student</p>
            <h1>Student Work</h1>

            <v-tabs
              v-model="studentTab"
              class="mt-6"
              color="primary"
              @update:model-value="handleStudentTabChange"
            >
              <v-tab value="answer">Answer Questions</v-tab>
              <v-tab value="grades">View Grades</v-tab>
            </v-tabs>

            <v-window v-model="studentTab" class="mt-6">
              <v-window-item value="answer">
                <v-progress-linear
                  v-if="isLoadingStudentAssignment"
                  class="mt-6"
                  color="primary"
                  indeterminate
                />

                <v-alert
                  v-else-if="!studentAssignment && !studentError"
                  class="mt-6"
                  type="success"
                  variant="tonal"
                >
                  You have finished all assignments
                </v-alert>

                <v-form
                  v-else-if="studentAssignment"
                  class="form-grid"
                  @submit.prevent="submitStudentAnswer"
                >
                  <div class="student-question">
                    <v-chip color="primary" size="small" variant="tonal">
                      {{ studentAssignment.category }}
                    </v-chip>
                    <p>{{ studentAssignment.prompt }}</p>
                  </div>

                  <v-textarea
                    v-model="studentAnswer"
                    label="Your answer"
                    rows="5"
                    variant="outlined"
                  />

                  <v-btn
                    :loading="isSubmittingStudentAnswer"
                    color="secondary"
                    size="large"
                    type="submit"
                  >
                    Submit Answer
                  </v-btn>
                </v-form>

                <section
                  v-if="studentAssignment && hasPreviousAttempts(studentAssignment)"
                  class="attempt-history"
                >
                  <h2>Previous Attempts</h2>
                  <v-expansion-panels class="mt-3" variant="accordion">
                    <v-expansion-panel
                      v-for="attempt in previousAttempts(studentAssignment)"
                      :key="attempt.id"
                    >
                      <v-expansion-panel-title>
                        <div class="question-title">
                          <v-chip size="small" variant="tonal">
                            Attempt {{ attempt.attempt_number }}
                          </v-chip>
                          <v-chip
                            :color="attemptStatusColor(attempt)"
                            size="small"
                            variant="tonal"
                          >
                            {{ attemptStatusLabel(attempt) }}
                          </v-chip>
                        </div>
                      </v-expansion-panel-title>
                      <v-expansion-panel-text>
                        <div class="review-grid">
                          <section class="review-box">
                            <h3>Your Answer</h3>
                            <p>{{ attempt.submitted_answer }}</p>
                          </section>
                          <section class="review-box submitted">
                            <h3>Feedback</h3>
                            <p>{{ attempt.feedback || 'No feedback yet.' }}</p>
                          </section>
                        </div>
                      </v-expansion-panel-text>
                    </v-expansion-panel>
                  </v-expansion-panels>
                </section>

                <v-alert v-if="studentMessage" class="mt-5" type="success" variant="tonal">
                  {{ studentMessage }}
                </v-alert>
                <v-alert v-if="studentError" class="mt-5" type="error" variant="tonal">
                  {{ studentError }}
                </v-alert>
              </v-window-item>

              <v-window-item value="grades">
                <div class="list-header">
                  <h2>Grade Summary</h2>
                  <v-btn
                    :loading="isLoadingStudentGrades"
                    color="primary"
                    prepend-icon="mdi-refresh"
                    variant="tonal"
                    @click="loadStudentGrades"
                  >
                    Refresh
                  </v-btn>
                </div>

                <v-progress-linear
                  v-if="isLoadingStudentGrades"
                  class="mt-3"
                  color="primary"
                  indeterminate
                />

                <v-alert v-if="studentGradesError" class="mt-5" type="error" variant="tonal">
                  {{ studentGradesError }}
                </v-alert>

                <div class="summary-grid">
                  <v-card
                    v-for="summary in gradeSummaries"
                    :key="summary.category"
                    class="summary-card"
                    variant="outlined"
                  >
                    <v-card-text>
                      <p class="summary-category">{{ summary.category }}</p>
                      <p class="summary-percent">{{ categoryPercentLabel(summary) }}</p>
                      <p class="summary-count">
                        {{ summary.passedCount }} of {{ summary.total }} passed
                      </p>
                    </v-card-text>
                  </v-card>
                </div>

                <v-alert
                  v-if="!isLoadingStudentGrades && !hasStudentGradedAssignments"
                  class="mt-5"
                  type="info"
                  variant="tonal"
                >
                  No graded assignments yet.
                </v-alert>

                <section
                  v-for="group in gradedAssignmentsByCategory"
                  v-else
                  :key="group.category"
                  class="graded-section"
                >
                  <h2>{{ group.category }}</h2>
                  <v-alert
                    v-if="group.assignments.length === 0"
                    class="mt-3"
                    type="info"
                    variant="tonal"
                  >
                    No graded {{ group.category.toLowerCase() }} assignments yet.
                  </v-alert>

                  <v-expansion-panels v-else class="mt-3" variant="accordion">
                    <v-expansion-panel
                      v-for="assignment in group.assignments"
                      :key="assignment.id"
                    >
                      <v-expansion-panel-title>
                        <div class="question-title">
                          <v-chip
                            :color="passFailColor(latestGradedAttempt(assignment)?.passed)"
                            size="small"
                            variant="tonal"
                          >
                            {{ assignment.category }}
                          </v-chip>
                          <v-chip
                            :color="passFailColor(latestGradedAttempt(assignment)?.passed)"
                            size="small"
                            variant="tonal"
                          >
                            {{ passFailLabel(latestGradedAttempt(assignment)?.passed) }}
                          </v-chip>
                          <span>{{ assignment.prompt }}</span>
                        </div>
                      </v-expansion-panel-title>
                      <v-expansion-panel-text>
                        <div
                          v-for="attempt in gradedAttempts(assignment)"
                          :key="attempt.id"
                          class="attempt-card"
                        >
                          <div class="question-title">
                            <v-chip size="small" variant="tonal">
                              Attempt {{ attempt.attempt_number }}
                            </v-chip>
                            <v-chip
                              :color="passFailColor(attempt.passed)"
                              size="small"
                              variant="tonal"
                            >
                              {{ passFailLabel(attempt.passed) }}
                            </v-chip>
                          </div>
                          <div class="review-grid mt-3">
                            <section class="review-box">
                              <h3>Your Answer</h3>
                              <p>{{ attempt.submitted_answer }}</p>
                            </section>
                            <section class="review-box">
                              <h3>Expected Answer</h3>
                              <p>{{ assignment.expected_answer }}</p>
                            </section>
                            <section class="review-box submitted">
                              <h3>Feedback</h3>
                              <p>{{ attempt.feedback || 'No feedback yet.' }}</p>
                            </section>
                          </div>
                        </div>
                      </v-expansion-panel-text>
                    </v-expansion-panel>
                  </v-expansion-panels>
                </section>
              </v-window-item>
            </v-window>
          </v-card-text>
        </v-card>

        <v-card v-else class="panel teacher-panel" elevation="8">
          <v-card-text>
            <div class="desk-header">
              <div>
                <p class="eyebrow">Teacher</p>
                <h1>Teacher Desk</h1>
              </div>
              <v-btn
                :loading="isLoggingOut"
                color="primary"
                prepend-icon="mdi-logout"
                variant="tonal"
                @click="logoutTeacher"
              >
                Logout
              </v-btn>
            </div>

            <section class="teacher-workspace">
              <v-btn
                :prepend-icon="isCreatingQuestion ? 'mdi-chevron-up' : 'mdi-plus'"
                color="secondary"
                variant="flat"
                @click="isCreatingQuestion = !isCreatingQuestion"
              >
                New Question
              </v-btn>

              <v-expand-transition>
                <v-form
                  v-if="isCreatingQuestion"
                  class="form-grid create-question-form"
                  @submit.prevent="saveAssignment"
                >
                  <v-select
                    v-model="form.category"
                    :items="categories"
                    label="Category"
                    variant="outlined"
                  />
                  <v-textarea
                    v-model="form.prompt"
                    label="Prompt"
                    placeholder="What is 7 + 5?"
                    rows="4"
                    variant="outlined"
                  />
                  <v-textarea
                    v-model="form.expected_answer"
                    label="Expected answer"
                    placeholder="12"
                    rows="3"
                    variant="outlined"
                  />
                  <v-btn
                    :loading="isSaving"
                    color="secondary"
                    prepend-icon="mdi-content-save"
                    size="large"
                    type="submit"
                  >
                    Save Assignment
                  </v-btn>
                </v-form>
              </v-expand-transition>

              <v-alert v-if="assignmentMessage" class="mt-5" type="success" variant="tonal">
                {{ assignmentMessage }}
              </v-alert>
              <v-alert v-if="assignmentError" class="mt-5" type="error" variant="tonal">
                {{ assignmentError }}
              </v-alert>
              <v-alert v-if="gradingMessage" class="mt-5" type="success" variant="tonal">
                {{ gradingMessage }}
              </v-alert>
              <v-alert v-if="gradingError" class="mt-5" type="error" variant="tonal">
                {{ gradingError }}
              </v-alert>

              <div class="list-header">
                <h2>Questions</h2>
                <v-btn
                  :loading="isLoadingAssignments"
                  color="primary"
                  prepend-icon="mdi-refresh"
                  variant="tonal"
                  @click="loadAssignments"
                >
                  Refresh
                </v-btn>
              </div>

              <v-btn-toggle
                v-model="teacherFilter"
                class="filter-toggle"
                color="primary"
                divided
                mandatory
                variant="outlined"
              >
                <v-btn
                  v-for="filter in teacherFilters"
                  :key="filter.value"
                  :value="filter.value"
                >
                  {{ filter.label }}
                </v-btn>
              </v-btn-toggle>

              <v-progress-linear
                v-if="isLoadingAssignments"
                class="mt-3"
                color="primary"
                indeterminate
              />

              <v-alert
                v-else-if="!hasAssignments"
                class="mt-4"
                type="info"
                variant="tonal"
              >
                No questions yet.
              </v-alert>

              <v-alert
                v-else-if="!hasFilteredAssignments"
                class="mt-4"
                type="info"
                variant="tonal"
              >
                No questions match this filter.
              </v-alert>

              <v-expansion-panels v-else class="mt-4" variant="accordion">
                <v-expansion-panel
                  v-for="assignment in filteredAssignments"
                  :key="assignment.id"
                >
                  <v-expansion-panel-title>
                    <div class="grading-title">
                      <div class="question-title">
                        <v-chip color="primary" size="small" variant="tonal">
                          {{ assignment.category }}
                        </v-chip>
                        <v-chip :color="statusColor(assignment)" size="small" variant="tonal">
                          {{ statusLabel(assignment) }}
                        </v-chip>
                        <v-chip v-if="currentAttempt(assignment)" size="small" variant="tonal">
                          Attempt {{ currentAttempt(assignment).attempt_number }}
                        </v-chip>
                        <span>{{ assignment.prompt }}</span>
                      </div>
                      <p v-if="currentAttempt(assignment)" class="answer-preview">
                        {{ answerPreview(currentAttempt(assignment).submitted_answer) }}
                      </p>
                    </div>
                  </v-expansion-panel-title>
                  <v-expansion-panel-text>
                    <div class="review-grid">
                      <section class="review-box">
                        <h3>Expected Answer</h3>
                        <p>{{ assignment.expected_answer }}</p>
                      </section>
                      <section v-if="currentAttempt(assignment)" class="review-box submitted">
                        <h3>Student Answer</h3>
                        <p>{{ currentAttempt(assignment).submitted_answer }}</p>
                      </section>
                    </div>

                    <v-form
                      v-if="currentAttempt(assignment)"
                      class="form-grid"
                      @submit.prevent="saveGrade(assignment)"
                    >
                      <v-btn-toggle
                        v-model="gradingForms[assignment.id].passed"
                        class="pass-fail-toggle"
                        color="primary"
                        mandatory
                        variant="outlined"
                      >
                        <v-btn
                          v-for="option in passFailOptions"
                          :key="option.value"
                          :color="option.value ? 'success' : 'error'"
                          :value="option.value"
                        >
                          {{ option.label }}
                        </v-btn>
                      </v-btn-toggle>
                      <v-textarea
                        v-model="gradingForms[assignment.id].feedback"
                        label="Feedback"
                        placeholder="Good explanation."
                        rows="3"
                        variant="outlined"
                      />
                      <div class="actions">
                        <v-btn
                          :loading="isSavingGrade"
                          color="secondary"
                          prepend-icon="mdi-content-save"
                          size="large"
                          type="submit"
                        >
                          Save Result
                        </v-btn>
                        <v-btn
                          :loading="isResettingAssignment"
                          color="warning"
                          prepend-icon="mdi-restore"
                          size="large"
                          variant="flat"
                          @click="resetAssignment(assignment)"
                        >
                          Reset Problem
                        </v-btn>
                      </div>
                    </v-form>

                    <section class="attempt-history">
                      <h2>Attempt History</h2>
                      <v-alert
                        v-if="attemptHistory(assignment).length === 0"
                        class="mt-3"
                        type="info"
                        variant="tonal"
                      >
                        No attempts yet.
                      </v-alert>
                      <div
                        v-for="attempt in attemptHistory(assignment)"
                        v-else
                        :key="attempt.id"
                        class="attempt-card"
                      >
                        <div class="question-title">
                          <v-chip size="small" variant="tonal">
                            Attempt {{ attempt.attempt_number }}
                          </v-chip>
                          <v-chip
                            :color="attemptStatusColor(attempt)"
                            size="small"
                            variant="tonal"
                          >
                            {{ attemptStatusLabel(attempt) }}
                          </v-chip>
                        </div>
                        <div class="review-grid mt-3">
                          <section class="review-box">
                            <h3>Student Answer</h3>
                            <p>{{ attempt.submitted_answer }}</p>
                          </section>
                          <section class="review-box submitted">
                            <h3>Feedback</h3>
                            <p>{{ attempt.feedback || 'No feedback yet.' }}</p>
                          </section>
                        </div>
                      </div>
                    </section>

                    <v-btn
                      class="mt-4"
                      color="error"
                      prepend-icon="mdi-delete"
                      variant="flat"
                      @click="deleteAssignment(assignment)"
                    >
                      Delete
                    </v-btn>
                  </v-expansion-panel-text>
                </v-expansion-panel>
              </v-expansion-panels>
            </section>
          </v-card-text>
        </v-card>
        <StudentPet v-if="view === 'student'" mood="idle" />
      </v-container>
    </v-main>
  </v-app>
</template>
