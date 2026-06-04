<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import FallingStarsGame from './components/pet/FallingStarsGame.vue'
import StudentPet from './components/StudentPet.vue'
import { useStudentPetStore } from './stores/studentPet'

const categories = ['MATH', 'SCIENCE', 'READING']
const studentCategoryOptions = [
  { title: 'All Subjects', value: 'ALL' },
  { title: 'Math', value: 'MATH' },
  { title: 'Science', value: 'SCIENCE' },
  { title: 'Reading', value: 'READING' },
]
const passFailOptions = [
  { label: 'Pass', value: true },
  { label: 'Fail', value: false },
]
const petStats = [
  { label: 'Hunger', key: 'hunger', color: 'warning', icon: 'mdi-food-apple' },
  { label: 'Happiness', key: 'happiness', color: 'success', icon: 'mdi-emoticon-happy' },
  { label: 'Energy', key: 'energy', color: 'primary', icon: 'mdi-lightning-bolt' },
]
const studentPetStore = useStudentPetStore()
const route = useRoute()
const router = useRouter()

const studentTab = ref('answer')
const studentCategoryFilter = ref('ALL')
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
const studentPetMood = ref('idle')
const isFallingStarsVisible = ref(false)
const lastFallingStarsResult = ref(null)
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
let studentPetMoodTimer

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
const routeName = computed(() => route.name || 'splash')
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
const canFeedPet = computed(() => studentPetStore.cookies > 0 && studentPetStore.hunger < 100)
const canPlayWithPet = computed(() => !studentPetStore.sleeping && studentPetStore.energy >= 10)
const canPutPetToSleep = computed(() => !studentPetStore.sleeping)
const canWakePet = computed(() => studentPetStore.sleeping)
const petAvatarMood = computed(() =>
  studentPetMood.value === 'idle' ? studentPetStore.mood : studentPetMood.value,
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

async function showSplash() {
  await router.push({ name: 'splash' })
}

async function showStudent() {
  studentTab.value = 'answer'
  studentCategoryFilter.value = 'ALL'
  await router.push({ name: 'student' })
}

async function showTeacherLogin() {
  await router.push({ name: 'teacher-login' })
}

async function handleRouteChange(name, oldName) {
  studentPetStore.stopWatching()
  resetStudentPetMood()

  if (name === 'splash') {
    loginError.value = ''
    return
  }

  if (name === 'student') {
    if (oldName !== 'student') {
      studentTab.value = 'answer'
      studentCategoryFilter.value = 'ALL'
    }
    await studentPetStore.loadProfile()
    studentPetStore.startWatching()
    if (studentTab.value === 'grades') {
      await loadStudentGrades()
    } else {
      await loadNextStudentAssignment()
    }
    return
  }

  if (name === 'teacher-login') {
    password.value = ''
    loginError.value = ''
    return
  }

  if (name === 'teacher') {
    teacherFilter.value = 'needs-review'
    await loadAssignments()
    if (assignmentError.value === 'teacher login required') {
      await router.replace({ name: 'teacher-login' })
    }
  }
}

watch(
  () => route.name,
  (name, oldName) => {
    handleRouteChange(name || 'splash', oldName)
  },
  { immediate: true },
)

async function handleStudentTabChange(tabName) {
  if (tabName === 'answer') {
    await loadNextStudentAssignment()
  }

  if (tabName === 'grades') {
    await studentPetStore.loadProfile()
    await loadStudentGrades()
  }

  if (tabName === 'pet') {
    await studentPetStore.loadProfile()
  }
}

async function handleStudentCategoryChange() {
  await loadNextStudentAssignment()
}

function resetStudentPetMood() {
  window.clearTimeout(studentPetMoodTimer)
  studentPetMood.value = 'idle'
}

function celebrateStudentAnswer() {
  window.clearTimeout(studentPetMoodTimer)
  studentPetMood.value = 'happy'
  studentPetMoodTimer = window.setTimeout(() => {
    studentPetMood.value = 'idle'
  }, 2600)
}

function playPetEatingAnimation() {
  window.clearTimeout(studentPetMoodTimer)
  studentPetMood.value = 'eating'
  studentPetMoodTimer = window.setTimeout(() => {
    studentPetMood.value = 'idle'
  }, 2600)
}

async function feedStudentPet() {
  const wasFed = await studentPetStore.feedPet()
  if (wasFed) {
    playPetEatingAnimation()
  }
}

function playWithStudentPet() {
  lastFallingStarsResult.value = null
  isFallingStarsVisible.value = true
}

async function handleFallingStarsComplete(result) {
  isFallingStarsVisible.value = false
  const wasApplied = await studentPetStore.applyGameResult(result)
  if (!wasApplied) {
    return
  }

  lastFallingStarsResult.value = result

  if (result.won) {
    celebrateStudentAnswer()
  }
}

function handleFallingStarsQuit() {
  isFallingStarsVisible.value = false
}

async function putStudentPetToSleep() {
  await studentPetStore.putToSleep()
}

async function wakeStudentPet() {
  await studentPetStore.wakePet()
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
    teacherFilter.value = 'needs-review'
    await router.push({ name: 'teacher' })
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
    await router.push({ name: 'splash' })
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
    const query = new URLSearchParams()
    if (studentCategoryFilter.value !== 'ALL') {
      query.set('category', studentCategoryFilter.value)
    }

    const endpoint = query.toString()
      ? `/api/student/assignments/next?${query.toString()}`
      : '/api/student/assignments/next'
    const response = await fetch(endpoint)
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

    await loadNextStudentAssignment()
    studentMessage.value = 'Answer submitted.'
    celebrateStudentAnswer()
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
        <v-card v-if="routeName === 'splash'" class="panel" elevation="8">
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

        <v-card v-else-if="routeName === 'teacher-login'" class="panel" elevation="8">
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

        <v-card v-else-if="routeName === 'student'" class="panel student-panel" elevation="8">
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
              <v-tab value="pet">Pet</v-tab>
            </v-tabs>

            <v-window v-model="studentTab" class="mt-6">
              <v-window-item value="answer">
                <v-select
                  v-model="studentCategoryFilter"
                  class="student-category-filter"
                  density="comfortable"
                  hide-details
                  item-title="title"
                  item-value="value"
                  :items="studentCategoryOptions"
                  label="Question category"
                  variant="outlined"
                  @update:model-value="handleStudentCategoryChange"
                />

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

              <v-window-item value="pet">
                <div class="list-header">
                  <h2>Pet</h2>
                  <v-btn
                    :loading="studentPetStore.isLoading"
                    color="primary"
                    prepend-icon="mdi-refresh"
                    variant="tonal"
                    @click="studentPetStore.loadProfile"
                  >
                    Refresh
                  </v-btn>
                </div>

                <v-alert v-if="studentPetStore.error" class="mt-5" type="error" variant="tonal">
                  {{ studentPetStore.error }}
                </v-alert>

                <section class="pet-dashboard">
                  <div class="cookie-display">
                    <span class="cookie-display__icon" aria-hidden="true" />
                    <div>
                      <p class="summary-category">Cookies</p>
                      <p class="cookie-display__count">{{ studentPetStore.cookies }}</p>
                    </div>
                    <div class="pet-actions">
                      <v-chip
                        :color="studentPetStore.sleeping ? 'primary' : 'success'"
                        size="small"
                        variant="tonal"
                      >
                        {{ studentPetStore.sleeping ? 'Sleeping' : 'Awake' }}
                      </v-chip>
                      <v-btn
                        :disabled="!canFeedPet"
                        :loading="studentPetStore.isLoading"
                        color="secondary"
                        prepend-icon="mdi-cookie"
                        variant="flat"
                        @click="feedStudentPet"
                      >
                        Feed Pet
                      </v-btn>
                      <v-btn
                        :disabled="!canPlayWithPet"
                        color="success"
                        prepend-icon="mdi-controller"
                        variant="tonal"
                        @click="playWithStudentPet"
                      >
                        Play
                      </v-btn>
                      <v-btn
                        :disabled="!canPutPetToSleep"
                        :loading="studentPetStore.isLoading"
                        color="primary"
                        prepend-icon="mdi-sleep"
                        variant="tonal"
                        @click="putStudentPetToSleep"
                      >
                        Sleep
                      </v-btn>
                      <v-btn
                        :disabled="!canWakePet"
                        :loading="studentPetStore.isLoading"
                        color="warning"
                        prepend-icon="mdi-weather-sunny"
                        variant="tonal"
                        @click="wakeStudentPet"
                      >
                        Wake
                      </v-btn>
                    </div>
                  </div>

                  <v-alert
                    v-if="studentPetStore.cookies === 0"
                    type="info"
                    variant="tonal"
                  >
                    Earn cookies by passing assignments.
                  </v-alert>
                  <v-alert
                    v-else-if="studentPetStore.hunger >= 100"
                    type="success"
                    variant="tonal"
                  >
                    Your pet is full.
                  </v-alert>

                  <v-alert
                    v-if="lastFallingStarsResult"
                    :type="lastFallingStarsResult.won ? 'success' : 'info'"
                    variant="tonal"
                  >
                    {{
                      lastFallingStarsResult.won
                        ? 'Great job! Your pet loved playing with the falling stars.'
                        : 'Almost! Catch more stars next time to make your pet happier.'
                    }}
                    Final score: {{ lastFallingStarsResult.score }} / 10.
                    Happiness gained: +{{ lastFallingStarsResult.happinessDelta }}.
                    Energy spent: {{ lastFallingStarsResult.energyDelta }}.
                  </v-alert>

                  <div class="pet-stat-list">
                    <div v-for="stat in petStats" :key="stat.key" class="pet-stat-row">
                      <div class="pet-stat-row__header">
                        <div class="question-title">
                          <v-icon :color="stat.color" :icon="stat.icon" size="small" />
                          <span>{{ stat.label }}</span>
                        </div>
                        <strong>{{ studentPetStore[stat.key] }}</strong>
                      </div>
                      <v-progress-linear
                        :color="stat.color"
                        :model-value="studentPetStore[stat.key]"
                        height="12"
                        rounded
                      />
                    </div>
                  </div>
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

        <div v-if="isFallingStarsVisible" class="game-overlay" role="dialog" aria-modal="true">
          <FallingStarsGame
            @complete="handleFallingStarsComplete"
            @quit="handleFallingStarsQuit"
          />
        </div>

        <StudentPet v-if="routeName === 'student'" :mood="petAvatarMood" />
      </v-container>
    </v-main>
  </v-app>
</template>
