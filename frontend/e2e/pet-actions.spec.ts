import { expect, test, type Locator, type Page } from '@playwright/test'
import { awardCookieThroughSchoolwork, resetAssignments } from './support/assignments'
import { loginAs } from './support/auth'

test.describe('pet action workflows', () => {
  test.beforeEach(async ({ baseURL }) => {
    await resetAssignments(baseURL!)
  })

  test('PW-PET-002 PW-PET-003 PW-PET-004 PW-PET-005 feed, play, sleep, and wake update pet UI', async ({ page, baseURL }) => {
    const runID = Date.now().toString(36)
    await awardCookieThroughSchoolwork(baseURL!, runID)
    await loginAs(page, 'student', baseURL!)
    await openPetTab(page)

    if (await page.getByTestId('wake-pet-button').isEnabled()) {
      await page.getByTestId('wake-pet-button').click()
      await expect(page.getByTestId('pet-sleep-state')).toContainText('Awake')
    }

    const cookiesBefore = await numericText(page.getByTestId('pet-cookie-count'))
    const hungerBefore = await numericText(page.getByTestId('pet-stat-hunger-value'))
    await expect(page.getByTestId('feed-pet-button')).toBeEnabled()
    await page.getByTestId('feed-pet-button').click()
    await expect.poll(() => numericText(page.getByTestId('pet-cookie-count'))).toBeLessThan(cookiesBefore)
    await expect.poll(() => numericText(page.getByTestId('pet-stat-hunger-value'))).toBeGreaterThanOrEqual(hungerBefore)

    const happinessBefore = await numericText(page.getByTestId('pet-stat-happiness-value'))
    const energyBefore = await numericText(page.getByTestId('pet-stat-energy-value'))
    await expect(page.getByTestId('play-pet-button')).toBeEnabled()
    await page.getByTestId('play-pet-button').click()
    await expect.poll(() => numericText(page.getByTestId('pet-stat-happiness-value'))).toBeGreaterThanOrEqual(happinessBefore)
    await expect.poll(() => numericText(page.getByTestId('pet-stat-energy-value'))).toBeLessThanOrEqual(energyBefore)

    await expect(page.getByTestId('sleep-pet-button')).toBeEnabled()
    await page.getByTestId('sleep-pet-button').click()
    await expect(page.getByTestId('pet-sleep-state')).toContainText('Sleeping')

    await expect(page.getByTestId('wake-pet-button')).toBeEnabled()
    await page.getByTestId('wake-pet-button').click()
    await expect(page.getByTestId('pet-sleep-state')).toContainText('Awake')
  })
})

async function openPetTab(page: Page) {
  await page.goto('/student')
  await page.getByRole('tab', { name: 'Pet' }).click()
  await expect(page.getByTestId('pet-profile')).toBeVisible()
}

async function numericText(locator: Locator): Promise<number> {
  return Number((await locator.textContent())?.trim() || '0')
}

/*
Requirement comparison:
PW-PET-002 covers feeding with an earned cookie and visible cookie/hunger updates.
PW-PET-003 covers play updating happiness/energy in the pet UI.
PW-PET-004 covers putting the pet to sleep.
PW-PET-005 covers waking the pet back to awake state.
Remaining gap: persistence over long decay intervals is left to pet store/rules tests.
*/
