import type { Category, StudentCategoryFilter } from '../types/assignment'
import type { CategoryOption, PetStat, TeacherFilterOption, ToggleOption } from '../types/ui'

export const categories = ['MATH', 'SCIENCE', 'READING'] as const satisfies readonly Category[]

export const studentCategoryOptions: CategoryOption[] = [
  { title: 'All Subjects', value: 'ALL' },
  { title: 'Math', value: 'MATH' },
  { title: 'Science', value: 'SCIENCE' },
  { title: 'Reading', value: 'READING' },
]

export const passFailOptions: ToggleOption<boolean>[] = [
  { label: 'Pass', value: true },
  { label: 'Fail', value: false },
]

export const petStats: PetStat[] = [
  { label: 'Hunger', key: 'hunger', color: 'warning', icon: 'mdi-food-apple' },
  { label: 'Happiness', key: 'happiness', color: 'success', icon: 'mdi-emoticon-happy' },
  { label: 'Energy', key: 'energy', color: 'primary', icon: 'mdi-lightning-bolt' },
]

export const teacherFilters: TeacherFilterOption[] = [
  { label: 'Needs Review', value: 'needs-review' },
  { label: 'All', value: 'all' },
  { label: 'Unanswered', value: 'unanswered' },
  { label: 'Reset', value: 'reset' },
  { label: 'Graded', value: 'graded' },
]

export const defaultStudentCategoryFilter: StudentCategoryFilter = 'ALL'
