import { test, expect, registerAndLogin } from '../../fixtures/auth'
import { getTomorrow } from '../../fixtures/data'

test.describe('Schedule Management', () => {
  test.beforeEach(async ({ page, testUser }) => {
    await registerAndLogin(page, testUser)
  })

  test('should display empty state when no schedules', async ({ schedulePage }) => {
    await schedulePage.goto()
    await schedulePage.waitForLoad()
    
    await expect(schedulePage.emptyMessage).toBeVisible()
    await expect(schedulePage.emptyMessage).toContainText('У вас пока нет настроенных расписаний')
  })

  test('should create recurring schedule', async ({ page, schedulePage }) => {
    await schedulePage.goto()
    await schedulePage.waitForLoad()
    
    await schedulePage.clickAdd()
    await schedulePage.fillScheduleForm({
      type: 'recurring',
      dayOfWeek: '1', // Monday
      startTime: '09:00',
      endTime: '17:00',
      isBlocked: false
    })
    await schedulePage.submitForm()
    
    // Wait for the schedule to appear in the table
    await page.waitForSelector('[data-testid^="schedule-row-"]')
    
    // Verify schedule is visible
    const scheduleRow = page.locator('[data-testid^="schedule-row-"]').first()
    await expect(scheduleRow).toBeVisible()
    await expect(page.locator('[data-testid^="schedule-type-"]').first()).toContainText('Повторяющееся')
    await expect(page.locator('[data-testid^="schedule-day-"]').first()).toContainText('Понедельник')
    await expect(page.locator('[data-testid^="schedule-time-"]').first()).toContainText('09:00 - 17:00')
  })

  test('should create one-time schedule', async ({ page, schedulePage }) => {
    const tomorrow = getTomorrow()
    
    await schedulePage.goto()
    await schedulePage.waitForLoad()
    
    await schedulePage.clickAdd()
    await schedulePage.fillScheduleForm({
      type: 'one-time',
      date: tomorrow,
      startTime: '10:00',
      endTime: '14:00',
      isBlocked: false
    })
    await schedulePage.submitForm()
    
    // Wait for the schedule to appear
    await page.waitForSelector('[data-testid^="schedule-row-"]')
    
    const scheduleRow = page.locator('[data-testid^="schedule-row-"]').first()
    await expect(scheduleRow).toBeVisible()
    await expect(page.locator('[data-testid^="schedule-type-"]').first()).toContainText('Разовое')
    await expect(page.locator('[data-testid^="schedule-time-"]').first()).toContainText('10:00 - 14:00')
  })

  test('should create blocked schedule', async ({ page, schedulePage }) => {
    await schedulePage.goto()
    await schedulePage.waitForLoad()
    
    await schedulePage.clickAdd()
    await schedulePage.fillScheduleForm({
      type: 'recurring',
      dayOfWeek: '6', // Saturday
      startTime: '00:00',
      endTime: '23:59',
      isBlocked: true
    })
    await schedulePage.submitForm()
    
    // Wait for the schedule to appear
    await page.waitForSelector('[data-testid^="schedule-row-"]')
    
    // Verify blocked status
    await expect(page.locator('[data-testid^="schedule-status-"]').first()).toContainText('Заблокировано')
  })

  test('should edit existing schedule', async ({ page, schedulePage }) => {
    // First create a schedule
    await schedulePage.goto()
    await schedulePage.waitForLoad()
    
    await schedulePage.clickAdd()
    await schedulePage.fillScheduleForm({
      type: 'recurring',
      dayOfWeek: '1',
      startTime: '09:00',
      endTime: '17:00'
    })
    await schedulePage.submitForm()
    
    // Wait for schedule to appear and get its ID
    await page.waitForSelector('[data-testid^="schedule-row-"]')
    const scheduleId = await page.locator('[data-testid^="schedule-row-"]').first().getAttribute('data-testid')
    const id = scheduleId?.replace('schedule-row-', '')
    
    // Edit the schedule
    await schedulePage.editSchedule(id!)
    await schedulePage.page.locator('[data-testid="schedule-start-time"]').fill('08:00')
    await schedulePage.page.locator('[data-testid="schedule-end-time"]').fill('16:00')
    await schedulePage.submitForm()
    
    // Verify updated time
    await expect(page.locator(`[data-testid="schedule-time-${id}"]`)).toContainText('08:00 - 16:00')
  })

  test('should delete schedule', async ({ page, schedulePage }) => {
    // Create a schedule first
    await schedulePage.goto()
    await schedulePage.waitForLoad()
    
    await schedulePage.clickAdd()
    await schedulePage.fillScheduleForm({
      type: 'recurring',
      dayOfWeek: '2',
      startTime: '09:00',
      endTime: '17:00'
    })
    await schedulePage.submitForm()
    
    // Wait for schedule to appear
    await page.waitForSelector('[data-testid^="schedule-row-"]')
    const scheduleRow = page.locator('[data-testid^="schedule-row-"]').first()
    await expect(scheduleRow).toBeVisible()
    
    // Get schedule ID and delete
    const scheduleId = await scheduleRow.getAttribute('data-testid')
    const id = scheduleId?.replace('schedule-row-', '')
    await schedulePage.deleteSchedule(id!)
    
    // Verify schedule is removed
    await expect(scheduleRow).not.toBeVisible()
  })

  test('should cancel form without saving', async ({ page, schedulePage }) => {
    await schedulePage.goto()
    await schedulePage.waitForLoad()
    
    await schedulePage.clickAdd()
    await schedulePage.fillScheduleForm({
      type: 'recurring',
      dayOfWeek: '1',
      startTime: '09:00',
      endTime: '17:00'
    })
    await schedulePage.cancelForm()
    
    // Verify no schedule was created
    await expect(schedulePage.emptyMessage).toBeVisible()
  })
})
