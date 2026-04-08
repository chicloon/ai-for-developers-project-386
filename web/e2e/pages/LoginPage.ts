import { Page, Locator, expect } from '@playwright/test'

export class LoginPage {
  readonly page: Page
  readonly emailInput: Locator
  readonly passwordInput: Locator
  readonly submitButton: Locator
  readonly errorMessage: Locator
  readonly registerLink: Locator

  constructor(page: Page) {
    this.page = page
    this.emailInput = page.locator('[data-testid="login-email-input"]')
    this.passwordInput = page.locator('[data-testid="login-password-input"]')
    this.submitButton = page.locator('[data-testid="login-submit-button"]')
    this.errorMessage = page.locator('[data-testid="login-error"]')
    this.registerLink = page.locator('[data-testid="login-register-link"]')
  }

  async goto() {
    await this.page.goto('/login')
    await expect(this.page.locator('[data-testid="login-page"]')).toBeVisible()
  }

  async login(email: string, password: string) {
    await this.emailInput.fill(email)
    await this.passwordInput.fill(password)
    await this.submitButton.click()
  }

  async expectError(message: string) {
    await expect(this.errorMessage).toContainText(message)
  }

  async clickRegister() {
    await this.registerLink.click()
  }
}
