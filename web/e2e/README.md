# E2E Tests for Call Booking

Playwright E2E тесты для приложения Call Booking.

## Структура

```
e2e/
├── fixtures/
│   ├── auth.ts          # Аутентификация и fixtures
│   └── data.ts          # Генерация тестовых данных
├── pages/
│   ├── LoginPage.ts     # Page Object для логина
│   ├── RegisterPage.ts  # Page Object для регистрации
│   ├── UsersPage.ts     # Page Object для каталога пользователей
│   └── SchedulePage.ts  # Page Object для расписания
└── specs/
    ├── auth/
    │   └── login.spec.ts       # Тесты аутентификации
    ├── features/
    │   └── schedule.spec.ts    # Тесты расписания
    └── smoke/
        └── health.spec.ts      # Smoke тесты
```

## Установка

```bash
# Установка зависимостей
npm install

# Установка браузеров Playwright
npx playwright install chromium
```

## Запуск тестов

```bash
# Запуск всех тестов
npm run test:e2e

# Запуск в UI режиме (для отладки)
npm run test:e2e:ui

# Запуск с дебагом
npm run test:e2e:debug

# Запуск только smoke тестов
npm run test:e2e:smoke

# Запуск только auth тестов
npm run test:e2e:auth

# Запуск feature тестов
npm run test:e2e:features
```

## Конфигурация

### Базовые настройки (playwright.config.ts)
- **Браузер**: Chromium
- **Параллельность**: Последовательное выполнение
- **Retries**: 2 в CI, 0 локально
- **Base URL**: http://localhost:3000

### Переменные окружения

```bash
# Использовать другой base URL
BASE_URL=http://localhost:3000 npm run test:e2e

# Запуск в CI режиме
CI=true npm run test:e2e
```

## Page Object Model

Тесты используют паттерн Page Object Model для лучшей поддерживаемости:

```typescript
// Пример использования
import { test, expect } from './fixtures/auth'
import { LoginPage } from './pages/LoginPage'

test('should login', async ({ page }) => {
  const loginPage = new LoginPage(page)
  await loginPage.goto()
  await loginPage.login('user@test.com', 'password')
  await expect(page).toHaveURL('/')
})
```

## Тестовые данные

Каждый тест создаёт уникального пользователя с timestamp:

```typescript
// Генерация тестового пользователя
const user = generateTestUser()
// { name: 'Test User 123456789', email: 'test123456789@example.com', password: 'TestPassword123!' }
```

## Data-testid атрибуты

Для стабильных селекторов используются data-testid атрибуты:

```tsx
// В React компонентах
<Button data-testid="login-submit-button">Войти</Button>

// В тестах
page.locator('[data-testid="login-submit-button"]')
```

## Отчёты

После выполнения тестов создаются отчёты:

```
playwright-report/
├── index.html          # HTML отчёт
└── data/
    └── *.zip          # Traces для отладки
```

Открыть отчёт:
```bash
npx playwright show-report
```

## Артефакты

При падении тестов сохраняются:
- Скриншоты (`screenshot: 'only-on-failure'`)
- Видео (`video: 'retain-on-failure'`)
- Traces (`trace: 'on-first-retry'`)

## Рекомендации

1. **Всегда используйте fixtures** для общих операций (login, setup)
2. **Используйте Page Objects** для сложных страниц
3. **Добавляйте data-testid** для критичных элементов
4. **Проверяйте URL** после навигации
5. **Используйте expect с await** для асинхронных проверок

## Добавление новых тестов

1. Создайте Page Object в `e2e/pages/` если нужно
2. Добавьте fixture в `e2e/fixtures/` если нужно
3. Создайте spec файл в `e2e/specs/`
4. Добавьте data-testid атрибуты в UI компоненты

## Отладка

```bash
# Запуск конкретного теста
npx playwright test specs/auth/login.spec.ts --grep "should login"

# Запуск с видимым браузером (headed mode)
npx playwright test --headed

# Запуск с паузой для отладки
npx playwright test --debug
```
