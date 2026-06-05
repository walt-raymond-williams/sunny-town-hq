import { computed, nextTick, reactive, ref, type ComponentPublicInstance } from 'vue'
import { getStudents } from '../api/studentsApi'
import {
  createAssignment,
  deleteAssignment as deleteAssignmentRequest,
  getAssignments,
  gradeAssignment,
  resetAssignment as resetAssignmentRequest,
} from '../api/teacherAssignmentsApi'
import { currentAttempt, attemptHistory } from '../domain/assignmentStatus'
import type {
  Assignment,
  CreateAssignmentPayload,
  GradingForm,
  TeacherFilter,
  TeacherStudentFilter,
} from '../types/assignment'
import type { StudentSummary } from '../types/user'
import type { SelectOption } from '../types/ui'

type AssignmentPanelRef = Element | ComponentPublicInstance
type AssignmentPanelRefs = Record<number, AssignmentPanelRef>

export function useTeacherAssignments() {
  const teacherFilter = ref<TeacherFilter>('needs-review')
  const teacherStudentFilter = ref<TeacherStudentFilter>('ALL')
  const isCreatingQuestion = ref(false)
  const assignmentMessage = ref('')
  const assignmentError = ref('')
  const gradingMessage = ref('')
  const gradingError = ref('')
  const assignments = ref<Assignment[]>([])
  const students = ref<StudentSummary[]>([])
  const gradingForms = reactive<Record<number, GradingForm>>({})
  const isLoadingAssignments = ref(false)
  const isSaving = ref(false)
  const isSavingGrade = ref(false)
  const isResettingAssignment = ref(false)
  const expandedAssignmentId = ref<number | null>(null)
  const assignmentPanelRefs = reactive<AssignmentPanelRefs>({})
  const form = ref<CreateAssignmentPayload>({
    category: 'MATH',
    prompt: '',
    expected_answer: '',
  })

  const teacherStudentOptions = computed<SelectOption<TeacherStudentFilter>[]>(() => [
    { title: 'All Students', value: 'ALL' },
    ...students.value.map((student) => ({
      title: student.display_name,
      value: String(student.id),
    })),
  ])
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

  async function loadStudents() {
    try {
      students.value = await getStudents()
    } catch (error) {
      assignmentError.value = errorMessage(error)
    }
  }

  async function loadAssignments() {
    isLoadingAssignments.value = true
    assignmentError.value = ''
    gradingError.value = ''

    try {
      assignments.value = await getAssignments(teacherStudentFilter.value)
      assignments.value.forEach((assignment) => {
        syncGradingForm(assignment)
      })
    } catch (error) {
      assignmentError.value = errorMessage(error)
      gradingError.value = errorMessage(error)
    } finally {
      isLoadingAssignments.value = false
    }
  }

  function setAssignmentPanelRef(assignmentID: number, element: AssignmentPanelRef | null) {
    if (element) {
      assignmentPanelRefs[assignmentID] = element
      return
    }

    delete assignmentPanelRefs[assignmentID]
  }

  async function handleExpandedAssignmentChange(assignmentID: number | null) {
    if (!assignmentID) {
      return
    }

    await nextTick()
    window.setTimeout(() => {
      const panelRef = assignmentPanelRefs[assignmentID]
      if (!panelRef) {
        return
      }
      const panel = '$el' in panelRef ? panelRef.$el : panelRef
      panel?.scrollIntoView({
        behavior: 'smooth',
        block: 'start',
      })
    }, 320)
  }

  async function saveAssignment() {
    isSaving.value = true
    assignmentMessage.value = ''
    assignmentError.value = ''

    try {
      await createAssignment({
        category: form.value.category,
        prompt: form.value.prompt.trim(),
        expected_answer: form.value.expected_answer.trim(),
      })

      assignmentMessage.value = 'Assignment saved.'
      form.value = {
        category: 'MATH',
        prompt: '',
        expected_answer: '',
      }
      isCreatingQuestion.value = false
      await loadAssignments()
    } catch (error) {
      assignmentError.value = errorMessage(error)
    } finally {
      isSaving.value = false
    }
  }

  async function deleteAssignment(assignment: Assignment) {
    assignmentMessage.value = ''
    assignmentError.value = ''

    try {
      await deleteAssignmentRequest(assignment.id)
      assignments.value = assignments.value.filter((item) => item.id !== assignment.id)
      assignmentMessage.value = 'Assignment deleted.'
      gradingMessage.value = ''
    } catch (error) {
      assignmentError.value = errorMessage(error)
    }
  }

  async function saveGrade(assignment: Assignment, passed = gradingForms[assignment.id]?.passed) {
    const gradeForm = gradingForms[assignment.id]
    if (!gradeForm) {
      return
    }

    gradeForm.passed = passed ?? null

    isSavingGrade.value = true
    gradingMessage.value = ''
    gradingError.value = ''

    try {
      await gradeAssignment(assignment.id, {
        attempt_id: currentAttempt(assignment)?.id,
        passed: gradeForm.passed,
        feedback: gradeForm.feedback.trim(),
      })

      gradingMessage.value = 'Result saved.'
      await loadAssignments()
    } catch (error) {
      gradingError.value = errorMessage(error)
    } finally {
      isSavingGrade.value = false
    }
  }

  async function resetAssignment(assignment: Assignment) {
    const gradeForm = gradingForms[assignment.id]
    isResettingAssignment.value = true
    gradingMessage.value = ''
    gradingError.value = ''

    try {
      await resetAssignmentRequest(assignment.id, {
        attempt_id: currentAttempt(assignment)?.id,
        feedback: gradeForm?.feedback?.trim() || '',
      })

      gradingMessage.value = 'Assignment reset. Previous attempts were kept.'
      await loadAssignments()
    } catch (error) {
      gradingError.value = errorMessage(error)
    } finally {
      isResettingAssignment.value = false
    }
  }

  async function handleTeacherStudentChange() {
    await loadAssignments()
  }

  function syncGradingForm(assignment: Assignment) {
    const attempt = currentAttempt(assignment)
    gradingForms[assignment.id] = {
      passed: attempt?.passed ?? null,
      feedback: attempt?.feedback || '',
    }
  }

  function resetTeacherWork() {
    teacherFilter.value = 'needs-review'
    teacherStudentFilter.value = 'ALL'
    assignments.value = []
    assignmentMessage.value = ''
    assignmentError.value = ''
    gradingMessage.value = ''
    gradingError.value = ''
  }

  return {
    assignmentError,
    assignmentMessage,
    assignments,
    deleteAssignment,
    expandedAssignmentId,
    filteredAssignments,
    form,
    gradingError,
    gradingForms,
    gradingMessage,
    handleExpandedAssignmentChange,
    handleTeacherStudentChange,
    hasAssignments,
    hasFilteredAssignments,
    isCreatingQuestion,
    isLoadingAssignments,
    isResettingAssignment,
    isSaving,
    isSavingGrade,
    loadAssignments,
    loadStudents,
    resetAssignment,
    resetTeacherWork,
    saveAssignment,
    saveGrade,
    setAssignmentPanelRef,
    teacherFilter,
    teacherStudentFilter,
    teacherStudentOptions,
  }
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
