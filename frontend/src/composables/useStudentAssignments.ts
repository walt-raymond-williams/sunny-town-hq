import { computed, ref } from 'vue'
import {
  getNextStudentAssignment,
  getStudentGradedAssignments,
  submitStudentAnswer as submitStudentAnswerRequest,
} from '../api/studentAssignmentsApi'
import { categories } from '../domain/categories'
import { gradedAttempts } from '../domain/assignmentStatus'
import type {
  Assignment,
  GradeSummary,
  GradedAssignmentGroup,
  StudentCategoryFilter,
} from '../types/assignment'

interface UseStudentAssignmentsOptions {
  onAnswerSubmitted?: () => void
}

export function useStudentAssignments({ onAnswerSubmitted }: UseStudentAssignmentsOptions = {}) {
  const studentTab = ref('answer')
  const studentCategoryFilter = ref<StudentCategoryFilter>('ALL')
  const studentGradedAssignments = ref<Assignment[]>([])
  const studentAssignment = ref<Assignment | null>(null)
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
  const gradeSummaries = computed<GradeSummary[]>(() =>
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
  const gradedAssignmentsByCategory = computed<GradedAssignmentGroup[]>(() =>
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
      studentError.value = errorMessage(error)
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
      studentGradesError.value = errorMessage(error)
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
      studentError.value = errorMessage(error)
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

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
