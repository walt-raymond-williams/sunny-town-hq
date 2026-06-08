import { expect, test } from '@playwright/test'
import { loginAs } from './support/auth'
import { hasNonBlankCanvasPixels } from './support/canvas'

test.describe('Sunny Town smoke workflow', () => {
  test('PW-ST-001 PW-ST-002 PW-ST-005 student enters Sunny Town, map renders, inventory toggles', async ({ page, baseURL }) => {
    await loginAs(page, 'student', baseURL!)
    await page.goto('/student')
    await page.getByRole('tab', { name: 'Pet' }).click()
    await expect(page.getByTestId('pet-profile')).toBeVisible()
    await page.getByTestId('enter-sunny-town-button').click()

    await expect(page.getByTestId('sunny-town-page')).toBeVisible()
    const canvas = page.getByTestId('sunny-town-canvas')
    await expect(canvas).toBeVisible()
    await expect.poll(() => hasNonBlankCanvasPixels(canvas)).toBe(true)

    await page.keyboard.press('E')
    await expect(page.getByTestId('sunny-town-inventory-panel')).toBeVisible()
    await page.keyboard.press('E')
    await expect(page.getByTestId('sunny-town-inventory-panel')).toBeHidden()
  })
})

/*
Requirement comparison:
PW-ST-001 covers entering Sunny Town from the student Pet page.
PW-ST-002 covers session startup, connected UI, canvas visibility, and a nonblank canvas pixel check after map rendering.
PW-ST-005 covers the keyboard inventory overlay open/close path.
Remaining gap: this smoke test does not validate precise movement, portals, NPC interaction, mining, or multiplayer; those are listed as later Playwright slices and partly covered by Go/Vitest now.
*/
