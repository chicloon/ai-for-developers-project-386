import { Page, Locator, expect } from '@playwright/test'

export class UsersPage {
  readonly page: Page
  readonly title: Locator
  readonly loadingIndicator: Locator
  readonly emptyMessage: Locator

  constructor(page: Page) {
    this.page = page
    this.title = page.locator('[data-testid="users-title"]')
    this.loadingIndicator = page.locator('[data-testid="users-loading"]')
    this.emptyMessage = page.locator('[data-testid="users-empty"]')
  }

  async goto() {
    await this.page.goto('/')
    await expect(this.page.locator('[data-testid="users-page"]')).toBeVisible()
  }

  async waitForLoad() {
    await this.loadingIndicator.waitFor({ state: 'hidden' })
  }

  async getUserCard(userId: string) {
    return this.page.locator(`[data-testid="user-card-${userId}"]`)
  }

  getUserName(userId: string): Locator {
    return this.page.locator(`[data-testid="user-name-${userId}"]`)
  }

  getUserEmail(userId: string): Locator {
    return this.page.locator(`[data-testid="user-email-${userId}"]`)
  }

  async clickBook(userId: string) {
    await this.page.locator(`[data-testid="user-book-button-${userId}"]`).click()
  }

  async expectUserVisible(userId: string, name: string, email: string) {
    const card = await this.getUserCard(userId)
    await expect(card).toBeVisible()
    await expect(this.getUserName(userId)).toContainText(name)
    await expect(this.getUserEmail(userId)).toContainText(email)
  }
}
