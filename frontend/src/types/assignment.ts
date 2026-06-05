export type Category = 'MATH' | 'SCIENCE' | 'READING'
export type StudentCategoryFilter = Category | 'ALL'
export type TeacherFilter = 'needs-review' | 'all' | 'unanswered' | 'reset' | 'graded'
export type TeacherStudentFilter = string | 'ALL'

export interface AssignmentAttempt {
  id: number
  assignment_id: number
  student_user_id: number
  student_display_name: string
  attempt_number: number
  submitted_answer: string
  date_submitted: string
  passed: boolean | null
  feedback: string | null
  date_graded: string | null
  cookie_awarded: boolean
  reset_at: string | null
}

export interface Assignment {
  id: number
  category: Category
  prompt: string
  expected_answer: string
  created_at: string
  current_attempt: AssignmentAttempt | null
  attempts: AssignmentAttempt[]
}

export interface NextStudentAssignmentResponse {
  assignment: Assignment | null
}

export interface CreateAssignmentPayload {
  category: Category
  prompt: string
  expected_answer: string
}

export interface SubmitAssignmentPayload {
  submitted_answer: string
}

export interface GradeAssignmentPayload {
  attempt_id: number | undefined
  passed: boolean | null
  feedback: string
}

export interface ResetAssignmentPayload {
  attempt_id: number | undefined
  feedback: string
}

export interface GradingForm {
  passed: boolean | null
  feedback: string
}

export interface GradeSummary {
  category: Category
  passedCount: number
  total: number
  percent: number | null
}

export interface GradedAssignmentGroup {
  category: Category
  assignments: Assignment[]
}
