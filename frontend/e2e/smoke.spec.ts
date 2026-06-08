import { expect, test } from '@playwright/test'
import { loginAs } from './support/auth'

test.describe('smoke coverage', () => {
  test('PW-SMOKE-001 REQ-FE-001 REQ-RUNTIME-001 app loads at root without browser errors', async ({ page }) => {
    const errors: string[] = []
    page.on('console', (message) => {
      if (message.type() === 'error') {
        errors.push(message.text())
      }
    })
    page.on('pageerror', (error) => errors.push(error.message))

    await page.goto('/')
    await expect(page).toHaveTitle('HQ')
    await expect(page.getByTestId('app-shell')).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Headquarters' })).toBeVisible()
    expect(errors).toEqual([])
  })

  test('PW-SMOKE-002 REQ-FE-001 deep routes refresh through the deployed app', async ({ page, baseURL }) => {
    await loginAs(page, 'student', baseURL!)
    await page.goto('/student')
    await expect(page.getByTestId('student-dashboard')).toBeVisible()

    await page.goto('/student/pet/sunny-town')
    await expect(page.getByTestId('sunny-town-page')).toBeVisible()

    const teacherPage = await page.context().newPage()
    await loginAs(teacherPage, 'teacher', baseURL!)
    await teacherPage.goto('/teacher')
    await expect(teacherPage.getByTestId('teacher-dashboard')).toBeVisible()

    await teacherPage.goto('/teacher/login')
    await expect(teacherPage).toHaveURL(/\/teacher$/)
    await expect(teacherPage.getByTestId('teacher-dashboard')).toBeVisible()
  })
})

/*
Requirement comparison:
PW-SMOKE-001 covers the deployed app shell at `/`, document title, and lack of browser console/page errors during initial render.
PW-SMOKE-002 covers SPA fallback and authenticated refresh behavior for student, teacher, teacher-login redirect, and Sunny Town routes.
Remaining gap: this smoke file does not prove unauthenticated Keycloak form entry; role and token-backed auth flows are covered in auth.spec.ts.
*/
