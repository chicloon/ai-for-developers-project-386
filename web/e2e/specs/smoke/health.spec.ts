import { test, expect } from '../../fixtures/auth'

test.describe('Smoke Tests', () => {
  test('should load login page', async ({ loginPage }) => {
    await loginPage.goto()
    await expect(loginPage.page.locator('[data-testid="login-page"]')).toBeVisible()
    await expect(loginPage.page.locator('[data-testid="login-title"]')).toContainText('Вход')
  })

  test('should load register page', async ({ registerPage }) => {
    await registerPage.goto()
    await expect(registerPage.page.locator('[data-testid="register-page"]')).toBeVisible()
    await expect(registerPage.page.locator('[data-testid="register-title"]')).toContainText('Регистрация')
  })

  test('should redirect to login when accessing protected route', async ({ page }) => {
    await page.goto('/my/schedule')
    await expect(page).toHaveURL('/login')
  })

  test('should load app layout after login', async ({ page, testUser }) => {
    const { registerAndLogin } = await import('../../fixtures/auth')
    await registerAndLogin(page, testUser)
    
    // Check navigation is visible
    await expect(page.locator('[data-testid="app-logo"]')).toBeVisible()
    await expect(page.locator('[data-testid="nav-users"]')).toBeVisible()
    await expect(page.locator('[data-testid="nav-schedule"]')).toBeVisible()
    await expect(page.locator('[data-testid="nav-groups"]')).toBeVisible()
    await expect(page.locator('[data-testid="nav-bookings"]')).toBeVisible()
  })

  test('should navigate between pages', async ({ page, testUser }) => {
    const { registerAndLogin } = await import('../../fixtures/auth')
    await registerAndLogin(page, testUser)
    
    // Navigate to schedule
    await page.locator('[data-testid="nav-schedule"]').click()
    await expect(page).toHaveURL('/my/schedule')
    
    // Navigate to groups
    await page.locator('[data-testid="nav-groups"]').click()
    await expect(page).toHaveURL('/my/groups')
    
    // Navigate to bookings
    await page.locator('[data-testid="nav-bookings"]').click()
    await expect(page).toHaveURL('/my/bookings')
    
    // Navigate back to users
    await page.locator('[data-testid="nav-users"]').click()
    await expect(page).toHaveURL('/')
  })
})
