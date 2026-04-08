import { Page, Locator, expect } from '@playwright/test'

export class UserProfilePage {
  readonly page: Page
  readonly title: Locator
  readonly loadingIndicator: Locator
  readonly noScheduleMessage: Locator
  readonly slotsContainer: Locator
  readonly datePicker: Locator

  constructor(page: Page) {
    this.page = page
    this.title = page.locator('[data-testid="user-profile-title"]')
    this.loadingIndicator = page.locator('[data-testid="user-profile-loading"]')
    this.noScheduleMessage = page.locator('[data-testid="user-profile-no-schedule"]')
    this.slotsContainer = page.locator('[data-testid="slots-container"]')
    this.datePicker = page.locator('[data-testid="slots-date-picker"]')
  }

  async goto(userId: string) {
    await this.page.goto(`/users/${userId}`)
    await expect(this.page.locator('[data-testid="user-profile-page"]')).toBeVisible()
  }

  async waitForLoad() {
    await this.loadingIndicator.waitFor({ state: 'hidden' })
  }

  async selectDate(date: string) {
    await this.datePicker.fill(date)
    await this.datePicker.press('Enter')
    // Wait for slots to load
    await this.page.waitForTimeout(500)
  }

  async getSlotButton(slotTime: string) {
    return this.page.locator(`[data-testid="slot-${slotTime}"]`)
  }

  async clickSlot(slotTime: string) {
    const slot = await this.getSlotButton(slotTime)
    await slot.click()
    await expect(this.page.locator('[data-testid="booking-modal"]')).toBeVisible()
  }

  async confirmBooking() {
    await this.page.locator('[data-testid="booking-confirm-button"]').click()
    await this.page.locator('[data-testid="booking-modal"]').waitFor({ state: 'hidden' })
  }

  async cancelBooking() {
    await this.page.locator('[data-testid="booking-cancel-button"]').click()
    await this.page.locator('[data-testid="booking-modal"]').waitFor({ state: 'hidden' })
  }

  async expectSlotVisible(slotTime: string) {
    const slot = await this.getSlotButton(slotTime)
    await expect(slot).toBeVisible()
  }

  async expectSlotBooked(slotTime: string) {
    await expect(this.page.locator(`[data-testid="slot-${slotTime}"]`)).toHaveAttribute(
      'data-booked',
      'true'
    )
  }

  async expectSlotAvailable(slotTime: string) {
    const slot = await this.getSlotButton(slotTime)
    await expect(slot).toBeVisible()
    await expect(slot).not.toHaveAttribute('data-booked', 'true')
    await expect(slot).not.toBeDisabled()
  }

  async expectUserName(name: string) {
    await expect(this.title).toContainText(name)
  }
}
