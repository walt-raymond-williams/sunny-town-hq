import { computed, ref } from 'vue'
import {
  getNextStudentAssignment,
  getStudentGradedAssignments,
  submitStudentAnswer as submitStudentAnswerRequest,
} from '../api/studentAssignmentsApi'
import { categories } from '../domain/categories'
import { gradedAttempts } from '../domain/assignmentStatus'

export function useStudentAssignments({ onAnswerSubmitted } = {}) {
  const studentTab = ref('answer')
  const studentCategoryFilter = ref('ALL')
  const studentGradedAssignments = ref([])
  const studentAssignment = ref(null)
  const studentAnswer = ref('')
  const studentMessage = ref('')
  const studentError = ref('')
  const studentGradesError = ref('')
  const isLoadingStudentAssignment = ref(false)
  const isLoadingStudentGrades = ref(false)
  const isSubmittingStudentAnswer = ref(false)

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

  async function loadNextStudentAssignment() {
    isLoadingStudentAssignment.value = true
    studentAssignment.value = null
    studentAnswer.value = ''
    studentMessage.value = ''
    studentError.value = ''

    try {
      studentAssignment.value = await getNextStudentAssignment(studentCategoryFilter.value)
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
      studentGradedAssignments.value = await getStudentGradedAssignments()
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
      await submitStudentAnswerRequest(studentAssignment.value.id, studentAnswer.value.trim())
      await loadNextStudentAssignment()
      studentMessage.value = 'Answer submitted.'
      onAnswerSubmitted?.()
    } catch (error) {
      studentError.value = error.message
    } finally {
      isSubmittingStudentAnswer.value = false
    }
  }

  async function handleStudentCategoryChange() {
    await loadNextStudentAssignment()
  }

  function resetStudentWork() {
    studentAssignment.value = null
    studentGradedAssignments.value = []
    studentAnswer.value = ''
    studentMessage.value = ''
    studentError.value = ''
    studentGradesError.value = ''
  }

  return {
    gradeSummaries,
    gradedAssignmentsByCategory,
    handleStudentCategoryChange,
    hasStudentGradedAssignments,
    isLoadingStudentAssignment,
    isLoadingStudentGrades,
    isSubmittingStudentAnswer,
    loadNextStudentAssignment,
    loadStudentGrades,
    resetStudentWork,
    studentAnswer,
    studentAssignment,
    studentCategoryFilter,
    studentError,
    studentGradesError,
    studentGradedAssignments,
    studentMessage,
    studentTab,
    submitStudentAnswer,
  }
}
