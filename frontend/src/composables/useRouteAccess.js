import { useRouter } from 'vue-router'
import { hasRole, keycloak, loginForRole } from '../auth'

export function useRouteAccess() {
  const router = useRouter()

  function authorizedHomeRoute() {
    if (hasRole('student')) {
      return 'student'
    }

    if (hasRole('teacher')) {
      return 'teacher'
    }

    return 'splash'
  }

  async function showSplash() {
    await router.push({ name: 'splash' })
  }

  async function showStudent() {
    if (keycloak.authenticated && !hasRole('student')) {
      await router.replace({ name: authorizedHomeRoute() })
      return false
    }

    if (!hasRole('student')) {
      await loginForRole('student', `${window.location.origin}/student`)
      return false
    }

    await router.push({ name: 'student' })
    return true
  }

  async function showTeacherLogin() {
    if (keycloak.authenticated && !hasRole('teacher')) {
      await router.replace({ name: authorizedHomeRoute() })
      return false
    }

    if (!hasRole('teacher')) {
      await loginForRole('teacher', `${window.location.origin}/teacher`)
      return false
    }

    await router.push({ name: 'teacher-login' })
    return true
  }

  async function ensureStudentAccess() {
    if (keycloak.authenticated && !hasRole('student')) {
      await router.replace({ name: authorizedHomeRoute() })
      return false
    }

    if (!hasRole('student')) {
      await loginForRole('student', `${window.location.origin}/student`)
      return false
    }

    return true
  }

  async function ensureTeacherAccess({ redirectToTeacher = false } = {}) {
    if (keycloak.authenticated && !hasRole('teacher')) {
      await router.replace({ name: authorizedHomeRoute() })
      return false
    }

    if (!hasRole('teacher')) {
      await loginForRole('teacher', `${window.location.origin}/teacher`)
      return false
    }

    if (redirectToTeacher) {
      await router.replace({ name: 'teacher' })
    }
    return true
  }

  return {
    authorizedHomeRoute,
    ensureStudentAccess,
    ensureTeacherAccess,
    showSplash,
    showStudent,
    showTeacherLogin,
  }
}
