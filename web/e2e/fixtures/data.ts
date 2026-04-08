export interface TestUser {
  name: string
  email: string
  password: string
}

export function generateTestUser(): TestUser {
  const timestamp = Date.now()
  return {
    name: `Test User ${timestamp}`,
    email: `test${timestamp}@example.com`,
    password: 'TestPassword123!'
  }
}

export function generateTestUsers(count: number): TestUser[] {
  return Array.from({ length: count }, (_, i) => {
    const timestamp = Date.now() + i
    return {
      name: `Test User ${timestamp}`,
      email: `test${timestamp}@example.com`,
      password: 'TestPassword123!'
    }
  })
}

export function formatDate(date: Date): string {
  return date.toISOString().split('T')[0]
}

export function formatTime(date: Date): string {
  return date.toTimeString().slice(0, 5)
}

export function getTomorrow(): string {
  const tomorrow = new Date()
  tomorrow.setDate(tomorrow.getDate() + 1)
  return formatDate(tomorrow)
}

export function getNextWeek(): string {
  const nextWeek = new Date()
  nextWeek.setDate(nextWeek.getDate() + 7)
  return formatDate(nextWeek)
}
