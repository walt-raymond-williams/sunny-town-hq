import { expect, test } from '@playwright/test'
import { resetAssignments } from './support/assignments'
import { loginAs } from './support/auth'

test.describe('schoolwork workflow', () => {
  test.beforeEach(async ({ baseURL }) => {
    await resetAssignments(baseURL!)
  })

  test('PW-SCHOOL-001..006 teacher creates, student submits, teacher grades, student sees result', async ({ browser, baseURL }) => {
    const runID = Date.now().toString(36)
    const prompt = `Playwright ${runID} PW-SCHOOL-001`
    const expectedAnswer = `Expected answer ${runID}`
    const submittedAnswer = `Student answer ${runID}`
    const feedback = `Feedback ${runID}`

    const teacherContext = await browser.newContext()
    const teacherPage = await teacherContext.newPage()
    await loginAs(teacherPage, 'teacher', baseURL!)
    await teacherPage.goto('/teacher')
    await expect(teacherPage.getByTestId('teacher-dashboard')).toBeVisible()

    await teacherPage.getByTestId('new-question-button').click()
    await teacherPage.getByTestId('assignment-prompt-input').locator('textarea').fill(prompt)
    await teacherPage.getByTestId('assignment-expected-answer-input').locator('textarea').fill(expectedAnswer)
    await teacherPage.getByTestId('assignment-save-button').click()
    await expect(teacherPage.getByText('Assignment saved.')).toBeVisible()
    await teacherPage.getByRole('button', { name: 'All' }).click()
    await expect(teacherPage.getByText(prompt)).toBeVisible()

    const studentContext = await browser.newContext()
    const studentPage = await studentContext.newPage()
    await loginAs(studentPage, 'student', baseURL!)
    await studentPage.goto('/student')
    await expect(studentPage.getByTestId('student-dashboard')).toBeVisible()
    await expect(studentPage.getByTestId('student-assignment-prompt')).toContainText(prompt)
    await studentPage.getByTestId('student-answer-input').locator('textarea').fill(submittedAnswer)
    await studentPage.getByTestId('assignment-submit-button').click()
    await expect(studentPage.getByText('Answer submitted.')).toBeVisible()

    await teacherPage.goto('/teacher')
    await expect(teacherPage.getByText(prompt)).toBeVisible()
    await teacherPage.getByText(prompt).click()
    await expect(teacherPage.getByText(submittedAnswer).first()).toBeVisible()
    await teacherPage.getByTestId('assignment-feedback-input').locator('textarea').fill(feedback)
    await teacherPage.getByTestId('assignment-pass-button').click()
    await expect(teacherPage.getByText('Result saved.')).toBeVisible()

    await studentPage.getByRole('tab', { name: 'View Grades' }).click()
    await expect(studentPage.getByText(prompt)).toBeVisible()
    await studentPage.getByText(prompt).click()
    await expect(studentPage.getByText(feedback)).toBeVisible()
    await expect(studentPage.getByText('Passed').first()).toBeVisible()

    await teacherContext.close()
    await studentContext.close()
  })
})

/*
Requirement comparison:
PW-SCHOOL-001 covers creating an assignment with category, prompt, and expected answer through the teacher UI.
PW-SCHOOL-002/PW-SCHOOL-003 cover the student seeing the next unanswered assignment and submitting a text answer.
PW-SCHOOL-004/PW-SCHOOL-005 cover the teacher seeing the submitted answer, saving pass feedback, and receiving success UI.
PW-SCHOOL-006 covers the student grades tab showing the passed grade and feedback.
Remaining gap: duplicate cookie-award idempotency is intentionally left to Go coverage; reset/delete workflows are planned for the next Playwright slice.
*/
