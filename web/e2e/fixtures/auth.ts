import { test as base, expect, Page } from '@playwright/test'
import { LoginPage } from '../pages/LoginPage'
import { RegisterPage } from '../pages/RegisterPage'
import { UsersPage } from '../pages/UsersPage'
import { SchedulePage } from '../pages/SchedulePage'
import { generateTestUser, TestUser } from './data'

// Extend base test with fixtures
type Fixtures = {
  loginPage: LoginPage
  registerPage: RegisterPage
  usersPage: UsersPage
  schedulePage: SchedulePage
  testUser: TestUser
}

export const test = base.extend<Fixtures>({
  loginPage: async ({ page }, use) => {
    await use(new LoginPage(page))
  },
  registerPage: async ({ page }, use) => {
    await use(new RegisterPage(page))
  },
  usersPage: async ({ page }, use) => {
    await use(new UsersPage(page))
  },
  schedulePage: async ({ page }, use) => {
    await use(new SchedulePage(page))
  },
  testUser: async ({}, use) => {
    await use(generateTestUser())
  },
})

export { expect }

// Helper function to register and login in one step
export async function registerAndLogin(page: Page, user: TestUser): Promise<void> {
  const registerPage = new RegisterPage(page)
  const usersPage = new UsersPage(page)

  await registerPage.goto()
  await registerPage.register(user.name, user.email, user.password)
  await expect(page).toHaveURL('/')
  await usersPage.waitForLoad()
}

// Helper function to login existing user
export async function loginUser(page: Page, user: TestUser): Promise<void> {
  const loginPage = new LoginPage(page)
  const usersPage = new UsersPage(page)

  await loginPage.goto()
  await loginPage.login(user.email, user.password)
  await expect(page).toHaveURL('/')
  await usersPage.waitForLoad()
}
