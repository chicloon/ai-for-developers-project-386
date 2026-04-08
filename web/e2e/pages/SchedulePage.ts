import { Page, Locator, expect } from '@playwright/test'

export class SchedulePage {
  readonly page: Page
  readonly title: Locator
  readonly addButton: Locator
  readonly loadingIndicator: Locator
  readonly emptyMessage: Locator
  readonly modal: Locator

  constructor(page: Page) {
    this.page = page
    this.title = page.locator('[data-testid="schedule-title"]')
    this.addButton = page.locator('[data-testid="schedule-add-button"]')
    this.loadingIndicator = page.locator('[data-testid="schedule-loading"]')
    this.emptyMessage = page.locator('[data-testid="schedule-empty"]')
    this.modal = page.locator('[data-testid="schedule-modal"]')
  }

  async goto() {
    await this.page.goto('/my/schedule')
    await expect(this.page.locator('[data-testid="schedule-page"]')).toBeVisible()
  }

  async waitForLoad() {
    await this.loadingIndicator.waitFor({ state: 'hidden' })
  }

  async clickAdd() {
    await this.addButton.click()
    // Wait for modal to open and check content is visible
    await expect(this.page.locator('[data-testid="schedule-type-select"]')).toBeVisible({ timeout: 5000 })
  }

  async fillScheduleForm(params: {
    type: 'recurring' | 'one-time'
    dayOfWeek?: string
    date?: string
    startTime: string
    endTime: string
    isBlocked?: boolean
  }) {
    // Select type - click on the select to open dropdown
    await this.page.locator('[data-testid="schedule-type-select"]').click()
    await this.page.waitForTimeout(300)
    // Click on the option by text in the dropdown (Mantine renders dropdown in portal)
    const typeLabel = params.type === 'recurring' ? 'Повторяющееся' : 'Разовое'
    await this.page.locator('.mantine-Select-option', { hasText: typeLabel }).click()
    // Wait for form to re-render based on type
    await this.page.waitForTimeout(400)

    if (params.type === 'recurring' && params.dayOfWeek) {
      await this.page.locator('[data-testid="schedule-day-select"]').click()
      await this.page.waitForTimeout(300)
      // Map day number to Russian name
      const dayNames: Record<string, string> = {
        '0': 'Воскресенье',
        '1': 'Понедельник',
        '2': 'Вторник',
        '3': 'Среда',
        '4': 'Четверг',
        '5': 'Пятница',
        '6': 'Суббота'
      }
      await this.page.locator('.mantine-Select-option', { hasText: dayNames[params.dayOfWeek] }).click()
    } else if (params.type === 'one-time' && params.date) {
      await this.page.locator('[data-testid="schedule-date-input"]').fill(params.date)
    }

    await this.page.locator('[data-testid="schedule-start-time"]').fill(params.startTime)
    await this.page.locator('[data-testid="schedule-end-time"]').fill(params.endTime)

    if (params.isBlocked) {
      await this.page.locator('[data-testid="schedule-blocked-checkbox"]').check()
    }
  }

  async submitForm() {
    await this.page.locator('[data-testid="schedule-submit-button"]').click()
    await this.modal.waitFor({ state: 'hidden' })
  }

  async cancelForm() {
    await this.page.locator('[data-testid="schedule-cancel-button"]').click()
    await this.modal.waitFor({ state: 'hidden' })
  }

  async getScheduleRow(scheduleId: string) {
    return this.page.locator(`[data-testid="schedule-row-${scheduleId}"]`)
  }

  async deleteSchedule(scheduleId: string) {
    await this.page.locator(`[data-testid="schedule-delete-${scheduleId}"]`).click()
  }

  async editSchedule(scheduleId: string) {
    await this.page.locator(`[data-testid="schedule-edit-${scheduleId}"]`).click()
    // Wait for modal to open and check content is visible
    await expect(this.page.locator('[data-testid="schedule-type-select"]')).toBeVisible({ timeout: 5000 })
  }

  async expectScheduleVisible(scheduleId: string, timeRange: string) {
    const row = await this.getScheduleRow(scheduleId)
    await expect(row).toBeVisible()
    await expect(this.page.locator(`[data-testid="schedule-time-${scheduleId}"]`)).toContainText(timeRange)
  }
}
