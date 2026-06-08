import { expect, test } from '@playwright/test'
import { loginAs } from './support/auth'

test.describe('pet and inventory smoke workflow', () => {
  test('PW-INV-001 PW-PET-001 student pet profile and inventory dialog load', async ({ page, baseURL }) => {
    await loginAs(page, 'student', baseURL!)
    await page.goto('/student')
    await page.getByRole('tab', { name: 'Pet' }).click()

    await expect(page.getByTestId('pet-profile')).toBeVisible()
    await expect(page.getByText('Cookies', { exact: true })).toBeVisible()
    await expect(page.getByText('Stars', { exact: true })).toBeVisible()
    await expect(page.getByText('Hunger')).toBeVisible()
    await expect(page.getByText('Happiness')).toBeVisible()
    await expect(page.getByText('Energy')).toBeVisible()
    await expect(page.getByText(/Awake|Sleeping/)).toBeVisible()

    await page.getByTestId('pet-inventory-button').click()
    await expect(page.getByTestId('pet-inventory-dialog')).toBeVisible()
    await expect(page.getByLabel('Equipment')).toBeVisible()
  })
})

/*
Requirement comparison:
PW-PET-001 covers visible pet cookie count, star count, sleep state, and core stats.
PW-INV-001 covers opening the Student Pet page inventory dialog and seeing equipment inventory content.
Remaining gap: feed/play/sleep/wake mutations and cross-view inventory consistency are deferred to the next Playwright slice.
*/
