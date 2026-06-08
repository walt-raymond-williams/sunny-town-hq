import { expect, test } from '@playwright/test'
import {
  type Assignment,
  createAssignment,
  resetAssignments,
  submitStudentAssignment,
} from './support/assignments'
import { hqFetch, loginAs } from './support/auth'

test.describe('schoolwork reset and delete workflows', () => {
  test.beforeEach(async ({ baseURL }) => {
    await resetAssignments(baseURL!)
  })

  test('PW-SCHOOL-007 PW-SCHOOL-008 teacher resets attempt and student sees prior history', async ({ browser, baseURL }) => {
    const runID = Date.now().toString(36)
    const prompt = `Playwright ${runID} PW-SCHOOL-007`
    const answer = `Reset answer ${runID}`
    const feedback = `Reset feedback ${runID}`
    const assignment = await createAssignment(baseURL!, {
      prompt,
      expectedAnswer: `Expected ${runID}`,
    })
    await submitStudentAssignment(baseURL!, assignment.id, answer)

    const teacherContext = await browser.newContext()
    const teacherPage = await teacherContext.newPage()
    await loginAs(teacherPage, 'teacher', baseURL!)
    await teacherPage.goto('/teacher')
    await expect(teacherPage.getByText(prompt)).toBeVisible()
    await teacherPage.getByText(prompt).click()
    await teacherPage.getByTestId('assignment-feedback-input').locator('textarea').fill(feedback)
    await teacherPage.getByTestId('assignment-reset-button').click()
    await expect(teacherPage.getByText('Assignment reset. Previous attempts were kept.')).toBeVisible()

    const studentContext = await browser.newContext()
    const studentPage = await studentContext.newPage()
    await loginAs(studentPage, 'student', baseURL!)
    await studentPage.goto('/student')
    await expect(studentPage.getByTestId('student-assignment-prompt')).toContainText(prompt)
    await expect(studentPage.getByText('Previous Attempts')).toBeVisible()
    await expect(studentPage.getByText(answer)).toBeVisible()
    await expect(studentPage.getByText(feedback)).toBeVisible()

    await teacherContext.close()
    await studentContext.close()
  })

  test('PW-SCHOOL-009 teacher deletes assignment and it disappears from teacher and student lists', async ({ browser, baseURL }) => {
    const runID = Date.now().toString(36)
    const prompt = `Playwright ${runID} PW-SCHOOL-009`
    await createAssignment(baseURL!, {
      prompt,
      expectedAnswer: `Expected ${runID}`,
    })

    const teacherContext = await browser.newContext()
    const teacherPage = await teacherContext.newPage()
    await loginAs(teacherPage, 'teacher', baseURL!)
    await teacherPage.goto('/teacher')
    await teacherPage.getByRole('button', { name: 'All' }).click()
    await expect(teacherPage.getByText(prompt)).toBeVisible()
    await teacherPage.getByText(prompt).click()
    await teacherPage.getByTestId('assignment-delete-button').click()
    await expect(teacherPage.getByText('Assignment deleted.')).toBeVisible()
    await expect(teacherPage.getByText(prompt)).toBeHidden()

    const assignments = await hqFetch<Assignment[]>('teacher', baseURL!, '/api/assignments')
    expect(assignments.some((assignment) => assignment.prompt === prompt)).toBe(false)

    const studentContext = await browser.newContext()
    const studentPage = await studentContext.newPage()
    await loginAs(studentPage, 'student', baseURL!)
    await studentPage.goto('/student')
    await expect(studentPage.getByText(prompt)).toBeHidden()
    await expect(studentPage.getByText('You have finished all assignments')).toBeVisible()

    await teacherContext.close()
    await studentContext.close()
  })
})

/*
Requirement comparison:
PW-SCHOOL-007 covers teacher reset of the latest active attempt with feedback through the UI.
PW-SCHOOL-008 covers the reset assignment becoming answerable again and previous answer/feedback appearing in student history.
PW-SCHOOL-009 covers teacher UI deletion, backend list removal, and student list disappearance.
Remaining gap: this browser test does not attempt deletion as a non-teacher; role rejection remains covered by Go/API auth tests.
*/
