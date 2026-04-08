import { Page, Locator, expect } from '@playwright/test'

export class BookingsPage {
  readonly page: Page
  readonly title: Locator
  readonly loadingIndicator: Locator
  readonly emptyMessage: Locator

  constructor(page: Page) {
    this.page = page
    this.title = page.locator('[data-testid="bookings-title"]')
    this.loadingIndicator = page.locator('[data-testid="bookings-loading"]')
    this.emptyMessage = page.locator('[data-testid="bookings-empty"]')
  }

  async goto() {
    await this.page.goto('/my/bookings')
    await expect(this.page.locator('[data-testid="bookings-page"]')).toBeVisible()
  }

  async waitForLoad() {
    await this.loadingIndicator.waitFor({ state: 'hidden' })
  }

  async getBookingRow(bookingId: string) {
    return this.page.locator(`[data-testid="booking-row-${bookingId}"]`)
  }

  async cancelBooking(bookingId: string) {
    await this.page.locator(`[data-testid="booking-cancel-${bookingId}"]`).click()
  }

  async expectBookingVisible(bookingId: string, ownerName?: string) {
    const row = await this.getBookingRow(bookingId)
    await expect(row).toBeVisible()
    if (ownerName) {
      await expect(this.page.locator(`[data-testid="booking-owner-${bookingId}"]`)).toContainText(ownerName)
    }
  }

  async expectBookingNotVisible(bookingId: string) {
    await expect(this.page.locator(`[data-testid="booking-row-${bookingId}"]`)).not.toBeVisible()
  }

  async expectBookingStatus(bookingId: string, status: 'active' | 'cancelled') {
    await expect(this.page.locator(`[data-testid="booking-status-${bookingId}"]`)).toContainText(
      status === 'active' ? 'Активна' : 'Отменена'
    )
  }
}
