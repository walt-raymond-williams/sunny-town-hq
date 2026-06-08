import { expect, test } from '@playwright/test'
import { loginAs } from './support/auth'

test.describe('auth and role boundaries', () => {
  test('PW-AUTH-001 REQ-AUTH-001 student token lands on student dashboard', async ({ page, baseURL }) => {
    await loginAs(page, 'student', baseURL!)
    await page.goto('/student')
    await expect(page.getByTestId('student-dashboard')).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Student Work' })).toBeVisible()
  })

  test('PW-AUTH-002 REQ-AUTH-001 teacher token lands on teacher dashboard', async ({ page, baseURL }) => {
    await loginAs(page, 'teacher', baseURL!)
    await page.goto('/teacher')
    await expect(page.getByTestId('teacher-dashboard')).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Teacher Desk' })).toBeVisible()
  })

  test('PW-AUTH-003 REQ-AUTH-001 student cannot access teacher page', async ({ page, baseURL }) => {
    await loginAs(page, 'student', baseURL!)
    await page.goto('/teacher')
    await expect(page).toHaveURL(/\/student$/)
    await expect(page.getByTestId('student-dashboard')).toBeVisible()
  })

  test('PW-AUTH-004 REQ-AUTH-001 STORY-ST-001 teacher cannot access student Sunny Town UI', async ({ page, baseURL }) => {
    await loginAs(page, 'teacher', baseURL!)
    await page.goto('/student/pet/sunny-town')
    await expect(page).toHaveURL(/\/teacher$/)
    await expect(page.getByTestId('teacher-dashboard')).toBeVisible()
  })
})

/*
Requirement comparison:
PW-AUTH-001/PW-AUTH-002 cover real Keycloak-issued bearer tokens, frontend role recognition, HQ authenticated API calls, and dashboard entry.
PW-AUTH-003/PW-AUTH-004 cover browser-visible role redirects for teacher-only and student-only routes.
Remaining gap: these tests use password-grant token seeding for stability instead of manually typing the Keycloak login form. Backend JWT validation and API role rejection remain covered by Go tests.
*/
