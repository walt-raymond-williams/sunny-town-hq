<script setup>
import { computed, onMounted, reactive, ref } from 'vue'

const categories = ['MATH', 'SCIENCE', 'READING']
const passFailOptions = [
  { label: 'Pass', value: true },
  { label: 'Fail', value: false },
]

const view = ref('splash')
const teacherTab = ref('grading')
const studentTab = ref('answer')
const password = ref('')
const loginError = ref('')
const assignmentMessage = ref('')
const assignmentError = ref('')
const gradingMessage = ref('')
const gradingError = ref('')
const assignments = ref([])
const answeredAssignments = ref([])
const gradingForms = reactive({})
const studentGradedAssignments = ref([])
const studentAssignment = ref(null)
const studentAnswer = ref('')
const studentMessage = ref('')
const studentError = ref('')
const studentGradesError = ref('')
const isLoadingAssignments = ref(false)
const isLoadingAnsweredAssignments = ref(false)
const isLoadingStudentAssignment = ref(false)
const isLoadingStudentGrades = ref(false)
const isSaving = ref(false)
const isSavingGrade = ref(false)
const isResettingAssignment = ref(false)
const isSubmittingStudentAnswer = ref(false)

const form = ref({
  category: 'MATH',
  prompt: '',
  expected_answer: '',
})

const hasAssignments = computed(() => assignments.value.length > 0)
const hasAnsweredAssignments = computed(() => answeredAssignments.value.length > 0)
const hasStudentGradedAssignments = computed(() => studentGradedAssignments.value.length > 0)
const gradeSummaries = computed(() =>
  categories.map((category) => {
    const assignmentsForCategory = studentGradedAssignments.value.filter(
      (assignment) => assignment.category === category,
    )
    const passedCount = assignmentsForCategory.filter((assignment) => assignment.passed).length
    const percent =
      assignmentsForCategory.length === 0
        ? null
        : Math.round((passedCount / assignmentsForCategory.length) * 100)

    return {
      category,
      passedCount,
      total: assignmentsForCategory.length,
      percent,
    }
  }),
)
const gradedAssignmentsByCategory = computed(() =>
  categories.map((category) => ({
    category,
    assignments: studentGradedAssignments.value.filter(
      (assignment) => assignment.category === category,
    ),
  })),
)

onMounted(() => {
  teacherTab.value = 'grading'
})

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
    teacherTab.value = 'grading'
    await loadAnsweredAssignments()
  } catch (error) {
    loginError.value = error.message
  }
}

async function handleTeacherTabChange(tabName) {
  if (tabName === 'grading') {
    await loadAnsweredAssignments()
  }

  if (tabName === 'create') {
    await loadAssignments()
  }
}

async function loadAssignments() {
  isLoadingAssignments.value = true
  assignmentError.value = ''

  try {
    const response = await fetch('/api/assignments')
    const body = await response.json()

    if (!response.ok) {
      throw new Error(body.error || `API returned ${response.status}`)
    }

    assignments.value = body
  } catch (error) {
    assignmentError.value = error.message
  } finally {
    isLoadingAssignments.value = false
  }
}

async function loadAnsweredAssignments() {
  isLoadingAnsweredAssignments.value = true
  gradingMessage.value = ''
  gradingError.value = ''

  try {
    const response = await fetch('/api/assignments/answered')
    const body = await response.json()

    if (!response.ok) {
      throw new Error(body.error || `API returned ${response.status}`)
    }

    answeredAssignments.value = body
    answeredAssignments.value.forEach((assignment) => {
      gradingForms[assignment.id] = {
        passed: assignment.passed,
        feedback: assignment.feedback || '',
      }
    })
  } catch (error) {
    gradingError.value = error.message
  } finally {
    isLoadingAnsweredAssignments.value = false
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
    await loadAnsweredAssignments()
  } catch (error) {
    gradingError.value = error.message
  } finally {
    isSavingGrade.value = false
  }
}

async function resetAssignment(assignment) {
  isResettingAssignment.value = true
  gradingMessage.value = ''
  gradingError.value = ''

  try {
    const response = await fetch(`/api/assignments/${assignment.id}/reset`, {
      method: 'PATCH',
    })
    const body = await response.json()

    if (!response.ok) {
      throw new Error(body.error || `API returned ${response.status}`)
    }

    gradingMessage.value = 'Assignment reset. Feedback was kept for the student.'
    await loadAnsweredAssignments()
  } catch (error) {
    gradingError.value = error.message
  } finally {
    isResettingAssignment.value = false
  }
}

function hasFeedback(assignment) {
  return Boolean(assignment?.feedback)
}

function statusLabel(assignment) {
  if (assignment.passed === true) {
    return 'Passed'
  }

  if (assignment.passed === false) {
    return 'Failed'
  }

  return 'Needs Review'
}

function statusColor(assignment) {
  if (assignment.passed === true) {
    return 'success'
  }

  if (assignment.passed === false) {
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

                  <section v-if="hasFeedback(studentAssignment)" class="feedback-callout">
                    <h3>Feedback</h3>
                    <p>{{ studentAssignment.feedback }}</p>
                  </section>

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
                            :color="passFailColor(assignment.passed)"
                            size="small"
                            variant="tonal"
                          >
                            {{ passFailLabel(assignment.passed) }}
                          </v-chip>
                          <span>{{ assignment.prompt }}</span>
                        </div>
                      </v-expansion-panel-title>
                      <v-expansion-panel-text>
                        <div class="review-grid">
                          <section class="review-box">
                            <h3>Your Answer</h3>
                            <p>{{ assignment.submitted_answer }}</p>
                          </section>
                          <section class="review-box">
                            <h3>Expected Answer</h3>
                            <p>{{ assignment.expected_answer }}</p>
                          </section>
                          <section class="review-box submitted">
                            <h3>Feedback</h3>
                            <p>{{ assignment.feedback || 'No feedback yet.' }}</p>
                          </section>
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
            <p class="eyebrow">Teacher</p>
            <h1>Teacher Desk</h1>

            <v-tabs
              v-model="teacherTab"
              class="mt-6"
              color="primary"
              @update:model-value="handleTeacherTabChange"
            >
              <v-tab value="grading">Grading Assignments</v-tab>
              <v-tab value="create">Creating Assignments</v-tab>
            </v-tabs>

            <v-window v-model="teacherTab" class="mt-6">
              <v-window-item value="grading">
                <div class="list-header">
                  <h2>Answered Questions</h2>
                  <v-btn
                    :loading="isLoadingAnsweredAssignments"
                    color="primary"
                    prepend-icon="mdi-refresh"
                    variant="tonal"
                    @click="loadAnsweredAssignments"
                  >
                    Refresh
                  </v-btn>
                </div>

                <v-alert v-if="gradingMessage" class="mt-5" type="success" variant="tonal">
                  {{ gradingMessage }}
                </v-alert>
                <v-alert v-if="gradingError" class="mt-5" type="error" variant="tonal">
                  {{ gradingError }}
                </v-alert>

                <v-progress-linear
                  v-if="isLoadingAnsweredAssignments"
                  class="mt-3"
                  color="primary"
                  indeterminate
                />

                <v-alert
                  v-else-if="!hasAnsweredAssignments"
                  class="mt-4"
                  type="info"
                  variant="tonal"
                >
                  No submitted assignments to grade yet.
                </v-alert>

                <v-expansion-panels v-else class="mt-4" variant="accordion">
                  <v-expansion-panel
                    v-for="assignment in answeredAssignments"
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
                          <span>{{ assignment.prompt }}</span>
                        </div>
                        <p class="answer-preview">
                          {{ answerPreview(assignment.submitted_answer) }}
                        </p>
                      </div>
                    </v-expansion-panel-title>
                    <v-expansion-panel-text>
                      <div class="review-grid">
                        <section class="review-box">
                          <h3>Prompt</h3>
                          <p>{{ assignment.prompt }}</p>
                        </section>
                        <section class="review-box">
                          <h3>Expected Answer</h3>
                          <p>{{ assignment.expected_answer }}</p>
                        </section>
                        <section class="review-box submitted">
                          <h3>Student Answer</h3>
                          <p>{{ assignment.submitted_answer }}</p>
                        </section>
                      </div>

                      <v-form class="form-grid" @submit.prevent="saveGrade(assignment)">
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
                      </v-form>
                    </v-expansion-panel-text>
                  </v-expansion-panel>
                </v-expansion-panels>
              </v-window-item>

              <v-window-item value="create">
                <v-form class="form-grid" @submit.prevent="saveAssignment">
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
                    size="large"
                    type="submit"
                  >
                    Save Assignment
                  </v-btn>
                </v-form>

                <v-alert v-if="assignmentMessage" class="mt-5" type="success" variant="tonal">
                  {{ assignmentMessage }}
                </v-alert>
                <v-alert v-if="assignmentError" class="mt-5" type="error" variant="tonal">
                  {{ assignmentError }}
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

                <v-expansion-panels v-else class="mt-4" variant="accordion">
                  <v-expansion-panel
                    v-for="assignment in assignments"
                    :key="assignment.id"
                  >
                    <v-expansion-panel-title>
                      <div class="question-title">
                        <v-chip color="primary" size="small" variant="tonal">
                          {{ assignment.category }}
                        </v-chip>
                        <span>{{ assignment.prompt }}</span>
                      </div>
                    </v-expansion-panel-title>
                    <v-expansion-panel-text>
                      <div class="answer-box">{{ assignment.expected_answer }}</div>
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
              </v-window-item>
            </v-window>
          </v-card-text>
        </v-card>
      </v-container>
    </v-main>
  </v-app>
</template>
