export type UserRole = 'teacher' | 'student' | string

export interface AuthUser {
  id: number
  keycloak_subject: string
  display_name: string
  email?: string
  roles: UserRole[]
}

export interface StudentSummary {
  id: number
  keycloak_subject?: string
  display_name: string
  email?: string
  roles?: UserRole[]
}
