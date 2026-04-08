import { test, expect, registerAndLogin, loginUser } from '../../fixtures/auth'

test.describe('Authentication', () => {
  test.describe('Registration', () => {
    test('should register new user successfully', async ({ page, registerPage, usersPage, testUser }) => {
      await registerPage.goto()
      await registerPage.register(testUser.name, testUser.email, testUser.password)
      
      // Should redirect to home page
      await expect(page).toHaveURL('/')
      await usersPage.waitForLoad()
      
      // Verify logged in state
      await expect(page.locator('[data-testid="user-name"]')).toContainText(testUser.name)
    })

    test('should show error for existing email', async ({ registerPage, testUser }) => {
      // First register
      await registerAndLogin(registerPage.page, testUser)
      
      // Try to register again with same email
      await registerPage.goto()
      await registerPage.register(testUser.name, testUser.email, testUser.password)
      
      await registerPage.expectError('user with this email already exists')
    })

    test('should validate required fields', async ({ registerPage }) => {
      await registerPage.goto()
      
      // Try to submit empty form
      await registerPage.submitButton.click()
      
      // Browser validation should prevent submission
      await expect(registerPage.page).toHaveURL('/register')
    })

    test('should navigate to login page', async ({ registerPage, loginPage }) => {
      await registerPage.goto()
      await registerPage.clickLogin()
      
      await expect(loginPage.page).toHaveURL('/login')
      await expect(loginPage.page.locator('[data-testid="login-page"]')).toBeVisible()
    })
  })

  test.describe('Login', () => {
    test('should login existing user', async ({ page, loginPage, usersPage, testUser }) => {
      // First register the user
      await registerAndLogin(page, testUser)
      
      // Logout
      await page.locator('[data-testid="user-menu"]').click()
      await page.locator('[data-testid="logout-button"]').click()
      
      // Login again
      await loginPage.goto()
      await loginPage.login(testUser.email, testUser.password)
      
      await expect(page).toHaveURL('/')
      await usersPage.waitForLoad()
      await expect(page.locator('[data-testid="user-name"]')).toContainText(testUser.name)
    })

    test('should show error for invalid credentials', async ({ loginPage, testUser }) => {
      await loginPage.goto()
      await loginPage.login(testUser.email, 'wrongpassword')
      
      await loginPage.expectError('invalid email or password')
    })

    test('should show error for non-existent user', async ({ loginPage }) => {
      await loginPage.goto()
      await loginPage.login('nonexistent@test.com', 'password123')
      
      await loginPage.expectError('invalid email or password')
    })

    test('should navigate to register page', async ({ loginPage, registerPage }) => {
      await loginPage.goto()
      await loginPage.clickRegister()
      
      await expect(registerPage.page).toHaveURL('/register')
      await expect(registerPage.page.locator('[data-testid="register-page"]')).toBeVisible()
    })
  })

  test.describe('Protected Routes', () => {
    test('should redirect unauthenticated user to login', async ({ page }) => {
      await page.goto('/my/schedule')
      await expect(page).toHaveURL('/login')
    })

    test('should redirect unauthenticated user from users page', async ({ page }) => {
      await page.goto('/')
      await expect(page).toHaveURL('/login')
    })

    test('should allow access after login', async ({ page, testUser }) => {
      await registerAndLogin(page, testUser)
      
      // Should be able to access protected routes
      await page.goto('/my/schedule')
      await expect(page.locator('[data-testid="schedule-page"]')).toBeVisible()
      
      await page.goto('/my/groups')
      await expect(page.locator('[data-testid="groups-page"]')).toBeVisible()
      
      await page.goto('/my/bookings')
      await expect(page.locator('[data-testid="bookings-page"]')).toBeVisible()
    })
  })

  test.describe('Logout', () => {
    test('should logout successfully', async ({ page, testUser }) => {
      await registerAndLogin(page, testUser)
      
      // Verify logged in
      await expect(page.locator('[data-testid="user-name"]')).toBeVisible()
      
      // Logout
      await page.locator('[data-testid="user-menu"]').click()
      await page.locator('[data-testid="logout-button"]').click()
      
      // Should redirect to login
      await expect(page).toHaveURL('/login')
    })
  })
})
