import { Page, Locator, expect } from '@playwright/test'

export class RegisterPage {
  readonly page: Page
  readonly nameInput: Locator
  readonly emailInput: Locator
  readonly passwordInput: Locator
  readonly submitButton: Locator
  readonly errorMessage: Locator
  readonly loginLink: Locator

  constructor(page: Page) {
    this.page = page
    this.nameInput = page.locator('[data-testid="register-name-input"]')
    this.emailInput = page.locator('[data-testid="register-email-input"]')
    this.passwordInput = page.locator('[data-testid="register-password-input"]')
    this.submitButton = page.locator('[data-testid="register-submit-button"]')
    this.errorMessage = page.locator('[data-testid="register-error"]')
    this.loginLink = page.locator('[data-testid="register-login-link"]')
  }

  async goto() {
    await this.page.goto('/register')
    await expect(this.page.locator('[data-testid="register-page"]')).toBeVisible()
  }

  async register(name: string, email: string, password: string) {
    await this.nameInput.fill(name)
    await this.emailInput.fill(email)
    await this.passwordInput.fill(password)
    await this.submitButton.click()
  }

  async expectError(message: string) {
    await expect(this.errorMessage).toContainText(message)
  }

  async clickLogin() {
    await this.loginLink.click()
  }
}
