import type { Category, StudentCategoryFilter, TeacherFilter } from './assignment'

export interface SelectOption<TValue extends string> {
  title: string
  value: TValue
}

export interface ToggleOption<TValue> {
  label: string
  value: TValue
}

export interface PetStat {
  label: string
  key: 'hunger' | 'happiness' | 'energy'
  color: 'warning' | 'success' | 'primary'
  icon: string
}

export interface TeacherFilterOption {
  label: string
  value: TeacherFilter
}

export type CategoryOption = SelectOption<StudentCategoryFilter>
export type CategoryValue = Category
