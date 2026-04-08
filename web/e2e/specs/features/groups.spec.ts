import { test, expect, registerAndLogin } from '../../fixtures/auth'
import { generateGroupName } from '../../fixtures/data'

test.describe('Group Management', () => {
  test.beforeEach(async ({ page, testUser }) => {
    await registerAndLogin(page, testUser)
  })

  test('should display empty state when no groups', async ({ groupsPage }) => {
    await groupsPage.goto()
    await groupsPage.waitForLoad()

    await expect(groupsPage.emptyMessage).toBeVisible()
    await expect(groupsPage.emptyMessage).toContainText('У вас пока нет созданных групп')
  })

  test('should create group with visibility level', async ({ page, groupsPage }) => {
    const groupName = generateGroupName()

    await groupsPage.goto()
    await groupsPage.waitForLoad()

    await groupsPage.clickAdd()
    await groupsPage.fillGroupForm({
      name: groupName,
      visibilityLevel: 'work'
    })
    await groupsPage.submitForm()

    // Wait for the group to appear in the table
    await page.waitForSelector('[data-testid^="group-row-"]')

    // Verify group is visible
    const groupRow = page.locator('[data-testid^="group-row-"]').first()
    await expect(groupRow).toBeVisible()
    await expect(page.locator('[data-testid^="group-name-"]').first()).toContainText(groupName)
    await expect(page.locator('[data-testid^="group-visibility-"]').first()).toContainText('Работа')
  })

  test('should create public group', async ({ page, groupsPage }) => {
    const groupName = generateGroupName()

    await groupsPage.goto()
    await groupsPage.waitForLoad()

    await groupsPage.clickAdd()
    await groupsPage.fillGroupForm({
      name: groupName,
      visibilityLevel: 'public'
    })
    await groupsPage.submitForm()

    await page.waitForSelector('[data-testid^="group-row-"]')
    await expect(page.locator('[data-testid^="group-visibility-"]').first()).toContainText('Публичная')
  })

  test('should create family group', async ({ page, groupsPage }) => {
    const groupName = generateGroupName()

    await groupsPage.goto()
    await groupsPage.waitForLoad()

    await groupsPage.clickAdd()
    await groupsPage.fillGroupForm({
      name: groupName,
      visibilityLevel: 'family'
    })
    await groupsPage.submitForm()

    await page.waitForSelector('[data-testid^="group-row-"]')
    await expect(page.locator('[data-testid^="group-visibility-"]').first()).toContainText('Семья')
  })

  test('should create friends group', async ({ page, groupsPage }) => {
    const groupName = generateGroupName()

    await groupsPage.goto()
    await groupsPage.waitForLoad()

    await groupsPage.clickAdd()
    await groupsPage.fillGroupForm({
      name: groupName,
      visibilityLevel: 'friends'
    })
    await groupsPage.submitForm()

    await page.waitForSelector('[data-testid^="group-row-"]')
    await expect(page.locator('[data-testid^="group-visibility-"]').first()).toContainText('Друзья')
  })

  test('should edit existing group', async ({ page, groupsPage }) => {
    const originalName = generateGroupName()
    const updatedName = generateGroupName()

    // Create group first
    await groupsPage.goto()
    await groupsPage.waitForLoad()

    await groupsPage.clickAdd()
    await groupsPage.fillGroupForm({
      name: originalName,
      visibilityLevel: 'work'
    })
    await groupsPage.submitForm()

    // Wait for group to appear and get its ID
    await page.waitForSelector('[data-testid^="group-row-"]')
    const groupId = await page.locator('[data-testid^="group-row-"]').first().getAttribute('data-testid')
    const id = groupId?.replace('group-row-', '')

    // Edit the group
    await groupsPage.editGroup(id!)
    await groupsPage.page.locator('[data-testid="group-name-input"]').fill(updatedName)

    // Change visibility level
    await groupsPage.page.locator('[data-testid="group-visibility-select"]').click()
    await groupsPage.page.waitForTimeout(300)
    await groupsPage.page.locator('.mantine-Select-option', { hasText: 'Семья' }).click()

    await groupsPage.submitForm()

    // Verify updated name and visibility
    await expect(page.locator(`[data-testid="group-name-${id}"]`)).toContainText(updatedName)
    await expect(page.locator(`[data-testid="group-visibility-${id}"]`)).toContainText('Семья')
  })

  test('should delete group', async ({ page, groupsPage }) => {
    const groupName = generateGroupName()

    // Create group first
    await groupsPage.goto()
    await groupsPage.waitForLoad()

    await groupsPage.clickAdd()
    await groupsPage.fillGroupForm({
      name: groupName,
      visibilityLevel: 'work'
    })
    await groupsPage.submitForm()

    // Wait for group to appear
    await page.waitForSelector('[data-testid^="group-row-"]')
    const groupRow = page.locator('[data-testid^="group-row-"]').first()
    await expect(groupRow).toBeVisible()

    // Get group ID and delete
    const groupId = await groupRow.getAttribute('data-testid')
    const id = groupId?.replace('group-row-', '')
    await groupsPage.deleteGroup(id!)

    // Verify group is removed
    await expect(groupRow).not.toBeVisible()
  })

  test('should add member by email', async ({ page, groupsPage, registerPage }) => {
    const groupName = generateGroupName()

    // First create another user to add as member
    const memberUser = {
      name: 'Member User',
      email: `member${Date.now()}@example.com`,
      password: 'TestPassword123!'
    }

    // Register member user in a new page context (but we need the same page)
    // Create member first
    await registerPage.goto()
    await registerPage.register(memberUser.name, memberUser.email, memberUser.password)

    // Wait for registration to complete and navigate back to groups
    await page.goto('/my/groups')
    await groupsPage.waitForLoad()

    // Create group
    await groupsPage.clickAdd()
    await groupsPage.fillGroupForm({
      name: groupName,
      visibilityLevel: 'friends'
    })
    await groupsPage.submitForm()

    // Wait for group to appear and get ID
    await page.waitForSelector('[data-testid^="group-row-"]')
    const groupId = await page.locator('[data-testid^="group-row-"]').first().getAttribute('data-testid')
    const id = groupId?.replace('group-row-', '')

    // Open members modal and add member
    await groupsPage.openMembers(id!)
    await groupsPage.addMemberByEmail(memberUser.email)

    // Verify member is added
    await expect(page.locator('[data-testid^="member-row-"]')).toBeVisible()
    await expect(page.locator('[data-testid^="member-email-"]').first()).toContainText(memberUser.email)

    // Close modal
    await groupsPage.closeMembersModal()
  })

  test('should remove member from group', async ({ page, groupsPage, registerPage }) => {
    const groupName = generateGroupName()

    // First create another user to add as member
    const memberUser = {
      name: 'Member To Remove',
      email: `removable${Date.now()}@example.com`,
      password: 'TestPassword123!'
    }

    // Register member user
    await registerPage.goto()
    await registerPage.register(memberUser.name, memberUser.email, memberUser.password)

    // Navigate back to groups
    await page.goto('/my/groups')
    await groupsPage.waitForLoad()

    // Create group
    await groupsPage.clickAdd()
    await groupsPage.fillGroupForm({
      name: groupName,
      visibilityLevel: 'friends'
    })
    await groupsPage.submitForm()

    // Wait for group to appear and get ID
    await page.waitForSelector('[data-testid^="group-row-"]')
    const groupId = await page.locator('[data-testid^="group-row-"]').first().getAttribute('data-testid')
    const id = groupId?.replace('group-row-', '')

    // Open members modal and add member
    await groupsPage.openMembers(id!)
    await groupsPage.addMemberByEmail(memberUser.email)

    // Wait for member to appear and get member ID
    await page.waitForSelector('[data-testid^="member-row-"]')
    const memberRow = page.locator('[data-testid^="member-row-"]').first()
    await expect(memberRow).toBeVisible()

    const memberId = await memberRow.getAttribute('data-testid')
    const mId = memberId?.replace('member-row-', '')

    // Remove member
    await groupsPage.removeMember(mId!)

    // Verify member is removed
    await expect(memberRow).not.toBeVisible()

    // Close modal
    await groupsPage.closeMembersModal()
  })

  test('should cancel form without saving', async ({ groupsPage }) => {
    const groupName = generateGroupName()

    await groupsPage.goto()
    await groupsPage.waitForLoad()

    await groupsPage.clickAdd()
    await groupsPage.fillGroupForm({
      name: groupName,
      visibilityLevel: 'work'
    })
    await groupsPage.cancelForm()

    // Verify no group was created
    await expect(groupsPage.emptyMessage).toBeVisible()
  })
})
