import { Page, Locator, expect } from '@playwright/test'

export class GroupsPage {
  readonly page: Page
  readonly title: Locator
  readonly addButton: Locator
  readonly loadingIndicator: Locator
  readonly emptyMessage: Locator
  readonly modal: Locator

  constructor(page: Page) {
    this.page = page
    this.title = page.locator('[data-testid="groups-title"]')
    this.addButton = page.locator('[data-testid="groups-add-button"]')
    this.loadingIndicator = page.locator('[data-testid="groups-loading"]')
    this.emptyMessage = page.locator('[data-testid="groups-empty"]')
    this.modal = page.locator('[data-testid="groups-modal"]')
  }

  async goto() {
    await this.page.goto('/my/groups')
    await expect(this.page.locator('[data-testid="groups-page"]')).toBeVisible()
  }

  async waitForLoad() {
    await this.loadingIndicator.waitFor({ state: 'hidden' })
  }

  async clickAdd() {
    await this.addButton.click()
    await expect(this.page.locator('[data-testid="group-name-input"]')).toBeVisible({ timeout: 5000 })
  }

  async fillGroupForm(params: {
    name: string
    visibilityLevel: 'family' | 'work' | 'friends' | 'public'
  }) {
    await this.page.locator('[data-testid="group-name-input"]').fill(params.name)

    // Select visibility level
    await this.page.locator('[data-testid="group-visibility-select"]').click()
    await this.page.waitForTimeout(300)

    // Map visibility level to Russian label
    const visibilityLabels: Record<string, string> = {
      family: 'Семья',
      work: 'Работа',
      friends: 'Друзья',
      public: 'Публичная'
    }
    await this.page.locator('.mantine-Select-option', { hasText: visibilityLabels[params.visibilityLevel] }).click()
  }

  async submitForm() {
    await this.page.locator('[data-testid="group-submit-button"]').click()
    await this.modal.waitFor({ state: 'hidden' })
  }

  async cancelForm() {
    await this.page.locator('[data-testid="group-cancel-button"]').click()
    await this.modal.waitFor({ state: 'hidden' })
  }

  async getGroupRow(groupId: string) {
    return this.page.locator(`[data-testid="group-row-${groupId}"]`)
  }

  async editGroup(groupId: string) {
    await this.page.locator(`[data-testid="group-edit-${groupId}"]`).click()
    await expect(this.page.locator('[data-testid="group-name-input"]')).toBeVisible({ timeout: 5000 })
  }

  async deleteGroup(groupId: string) {
    await this.page.locator(`[data-testid="group-delete-${groupId}"]`).click()
  }

  async openMembers(groupId: string) {
    await this.page.locator(`[data-testid="group-members-${groupId}"]`).click()
    await expect(this.page.locator('[data-testid="members-modal"]')).toBeVisible({ timeout: 5000 })
  }

  async addMemberByEmail(email: string) {
    await this.page.locator('[data-testid="member-email-input"]').fill(email)
    await this.page.locator('[data-testid="member-add-button"]').click()
  }

  async removeMember(memberId: string) {
    await this.page.locator(`[data-testid="member-remove-${memberId}"]`).click()
  }

  async closeMembersModal() {
    await this.page.locator('[data-testid="members-close-button"]').click()
    await this.page.locator('[data-testid="members-modal"]').waitFor({ state: 'hidden' })
  }

  async expectGroupVisible(groupId: string, name: string, visibility: string) {
    const row = await this.getGroupRow(groupId)
    await expect(row).toBeVisible()
    await expect(this.page.locator(`[data-testid="group-name-${groupId}"]`)).toContainText(name)
    await expect(this.page.locator(`[data-testid="group-visibility-${groupId}"]`)).toContainText(visibility)
  }

  async expectMemberVisible(memberId: string, name: string, email: string) {
    await expect(this.page.locator(`[data-testid="member-row-${memberId}"]`)).toBeVisible()
    await expect(this.page.locator(`[data-testid="member-name-${memberId}"]`)).toContainText(name)
    await expect(this.page.locator(`[data-testid="member-email-${memberId}"]`)).toContainText(email)
  }
}
