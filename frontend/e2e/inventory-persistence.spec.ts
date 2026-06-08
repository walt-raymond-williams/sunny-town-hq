import { expect, test, type Page } from '@playwright/test'
import { loginAs } from './support/auth'

test.describe('inventory equipment persistence', () => {
  test('PW-INV-004 PW-INV-005 equip and unequip persists across refresh', async ({ page, baseURL }) => {
    await loginAs(page, 'student', baseURL!)
    await openPetInventory(page)

    const unequipGear = page.getByTestId('unequip-gear-button')
    if (await unequipGear.isVisible()) {
      await unequipGear.click()
      await expect(page.getByTestId('inventory-item-sunny_hoodie')).toBeVisible()
    }

    await page.getByTestId('equip-sunny_hoodie-button').click()
    await expect(page.getByTestId('equipment-slot-gear')).toContainText('Sunny Hoodie')

    await page.reload()
    await openPetInventory(page)
    await expect(page.getByTestId('equipment-slot-gear')).toContainText('Sunny Hoodie')

    await page.getByTestId('unequip-gear-button').click()
    await expect(page.getByTestId('inventory-item-sunny_hoodie')).toBeVisible()

    await page.reload()
    await openPetInventory(page)
    await expect(page.getByTestId('inventory-item-sunny_hoodie')).toBeVisible()
  })
})

async function openPetInventory(page: Page) {
  await page.goto('/student')
  await page.getByRole('tab', { name: 'Pet' }).click()
  await page.getByTestId('pet-inventory-button').click()
  await expect(page.getByTestId('pet-inventory-dialog')).toBeVisible()
}

/*
Requirement comparison:
PW-INV-004 covers equipping owned gear and verifying the equipped state after route reload.
PW-INV-005 covers unequipping gear and verifying the unequipped item remains visible after route reload.
Remaining gap: wrong-slot and unowned/non-equippable rejection are lower-level API coverage, not browser coverage here.
*/
