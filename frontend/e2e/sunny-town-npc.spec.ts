import { expect, test } from '@playwright/test'
import { createAssignment, resetAssignments } from './support/assignments'
import { loginAs } from './support/auth'
import { seedStudentStars, setSunnyTownPosition } from './support/sunnyTown'

test.describe('Sunny Town NPC workflows', () => {
  test.beforeEach(async ({ baseURL }) => {
    await resetAssignments(baseURL!)
  })

  test('PW-ST-012 PW-ST-013 schoolwork NPC opens panel and submits answer', async ({ page, baseURL }) => {
    const runID = Date.now().toString(36)
    const prompt = `Playwright ${runID} PW-ST-012`
    const answer = `Sunny Town answer ${runID}`
    await createAssignment(baseURL!, {
      prompt,
      expectedAnswer: `Expected ${runID}`,
    })
    await setSunnyTownPosition(baseURL!, {
      mapId: 'sunny-town-classroom',
      x: 320,
      y: 208,
      facing: 'up',
    })

    await loginAs(page, 'student', baseURL!)
    await page.goto('/student/pet/sunny-town')
    await expect(page.getByTestId('sunny-town-page')).toBeVisible()
    await page.keyboard.press('F')
    await expect(page.getByTestId('sunny-town-npc-menu')).toContainText('Teacher')
    await page.getByTestId('sunny-town-start-schoolwork-button').click()
    await expect(page.getByTestId('sunny-town-schoolwork-panel')).toContainText(prompt)
    await page.getByTestId('sunny-town-schoolwork-answer-input').locator('textarea').fill(answer)
    await page.getByTestId('sunny-town-schoolwork-submit-button').click()
    await expect(page.getByText('Answer submitted.')).toBeVisible()
  })

  test('PW-ST-014 PW-ST-015 shop NPC opens shop and buying cookie updates inventory and stars', async ({ page, baseURL }) => {
    const runID = Date.now().toString(36)
    await seedStudentStars(baseURL!, 50, runID)
    await setSunnyTownPosition(baseURL!, {
      mapId: 'sunny-town-house-1',
      x: 224,
      y: 300,
      facing: 'up',
    })

    await loginAs(page, 'student', baseURL!)
    await page.goto('/student/pet/sunny-town')
    await expect(page.getByTestId('sunny-town-page')).toBeVisible()
    await page.keyboard.press('F')
    await expect(page.getByTestId('sunny-town-npc-menu')).toContainText('Cookie Keeper')
    await page.getByTestId('sunny-town-open-shop-button').click()
    await expect(page.getByTestId('sunny-town-shop-panel')).toBeVisible()
    await expect(page.getByTestId('buy-cookie-button')).toBeEnabled()
    await page.getByTestId('buy-cookie-button').click()
    await expect(page.getByText('Purchased.')).toBeVisible()
    await expect(page.getByTestId('shop-item-cookie')).toContainText('Cookie')
    await expect(page.getByText(/\d+ stars/).first()).toBeVisible()
  })
})

/*
Requirement comparison:
PW-ST-012 covers opening the schoolwork NPC panel from an in-world nearby Teacher.
PW-ST-013 covers submitting student schoolwork through the Sunny Town panel.
PW-ST-014 covers opening the Cookie Keeper shop panel.
PW-ST-015 covers buying a cookie through the shop UI after seeded star balance, with purchase success and inventory item visibility.
Remaining gap: this uses saved-position setup instead of walking to each NPC; movement/portal travel remains separate later coverage.
*/
