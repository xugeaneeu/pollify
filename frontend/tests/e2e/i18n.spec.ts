import { expect, test, type Page } from '@playwright/test'
import { resetDb } from './helpers/db'
import { uniqueEmail, futureLocal, pastLocal } from './helpers/factories'

test.beforeEach(() => {
  resetDb()
})

async function switchTo(page: Page, label: 'EN' | 'RU') {
  await page.getByRole('button', { name: label, exact: true }).click()
}

async function registerInRu(page: Page, email: string) {
  await page.goto('/register')
  await switchTo(page, 'RU')
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Имя для показа').fill('Иван')
  await page.getByLabel('Пароль').fill('password123')
  await page.getByRole('button', { name: 'Зарегистрироваться', exact: true }).click()
  await expect(page).toHaveURL(/\/polls$/)
}

test('default locale is English', async ({ page }) => {
  await page.goto('/login')
  await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible()
})

test('language switcher flips strings on the login screen', async ({ page }) => {
  await page.goto('/login')
  await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible()

  await switchTo(page, 'RU')
  await expect(page.getByRole('heading', { name: 'С возвращением' })).toBeVisible()
  await expect(page.getByLabel('Пароль')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Войти', exact: true })).toBeVisible()

  await switchTo(page, 'EN')
  await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible()
})

test('locale choice persists across reload', async ({ page }) => {
  await page.goto('/login')
  await switchTo(page, 'RU')
  await expect(page.getByRole('heading', { name: 'С возвращением' })).toBeVisible()

  await page.reload()
  await expect(page.getByRole('heading', { name: 'С возвращением' })).toBeVisible()

  await page.goto('/register')
  await expect(page.getByRole('heading', { name: 'Создать аккаунт' })).toBeVisible()
})

test('topbar and poll list are translated in Russian after RU registration', async ({ page }) => {
  await registerInRu(page, uniqueEmail('ru'))

  await expect(page.getByRole('link', { name: 'Опросы' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Новый опрос', exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Выйти' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Опросы' })).toBeVisible()

  // Filter chips
  await expect(page.getByRole('button', { name: 'Все', exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Активные' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Запланированные' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Завершённые' })).toBeVisible()

  // Filter toggles
  await expect(page.getByText('Доступные сейчас')).toBeVisible()
  await expect(page.getByText('Только анонимные')).toBeVisible()
  await expect(page.getByText('Мои опросы')).toBeVisible()
})

test('create poll form labels are in Russian', async ({ page }) => {
  await registerInRu(page, uniqueEmail('ru'))
  await page.getByRole('link', { name: 'Новый опрос', exact: true }).click()

  await expect(page.getByRole('heading', { name: 'Создать опрос' })).toBeVisible()
  await expect(page.getByLabel('Название')).toBeVisible()
  await expect(page.getByLabel('Вопрос')).toBeVisible()
  await expect(page.getByLabel('Анонимное голосование')).toBeVisible()
  await expect(page.getByLabel(/Свободные текстовые ответы/)).toBeVisible()
  await expect(page.getByLabel('Множественный выбор')).toBeVisible()
  await expect(page.getByLabel('Начало')).toBeVisible()
  await expect(page.getByLabel('Окончание')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Создать опрос' })).toBeVisible()
})

test('Russian pluralization: 0 vs 1 participant on the poll list', async ({ page }) => {
  await registerInRu(page, uniqueEmail('ru'))

  // Create a poll with the form in Russian
  await page.getByRole('link', { name: 'Новый опрос', exact: true }).click()
  await page.getByLabel('Название').fill('Обед?')
  await page.getByLabel('Вопрос').fill('Где обедаем')
  await page.getByLabel('Варианты (по одному в строке)').fill('Пицца\nСуши')
  await page.getByLabel('Начало').fill(pastLocal(60))
  await page.getByLabel('Окончание').fill(futureLocal(60 * 24))
  await page.getByRole('button', { name: 'Создать опрос' }).click()
  await expect(page).toHaveURL(/\/polls\/[0-9a-f-]+$/)

  // List page shows "0 участников" (many form for 0 in Russian)
  await page.getByRole('link', { name: 'Опросы' }).click()
  await expect(page.getByText('0 участников')).toBeVisible()

  // Cast one vote → "1 участник" (one form)
  await page.getByRole('heading', { name: 'Обед?' }).click()
  await page.getByRole('button', { name: 'Пицца' }).click()
  await page.getByRole('button', { name: 'Отправить голос' }).click()
  await expect(page).toHaveURL(/\/results$/)

  await page.getByRole('link', { name: 'Назад к опросу' }).click()
  await expect(page.getByText('1 участник', { exact: false })).toBeVisible()
  await page.getByRole('link', { name: 'Ко всем опросам' }).click()
  await expect(page.getByText('1 участник')).toBeVisible()
  await expect(page.getByText('Вы голосовали')).toBeVisible()
})

test('status tags use Russian labels', async ({ page }) => {
  await registerInRu(page, uniqueEmail('ru'))
  await page.getByRole('link', { name: 'Новый опрос', exact: true }).click()
  await page.getByLabel('Название').fill('Опрос для теста')
  await page.getByLabel('Вопрос').fill('Тест')
  await page.getByLabel('Варианты (по одному в строке)').fill('Да\nНет')
  await page.getByLabel('Начало').fill(pastLocal(60))
  await page.getByLabel('Окончание').fill(futureLocal(60 * 24))
  await page.getByRole('button', { name: 'Создать опрос' }).click()

  // The status tag on the detail page should say "активный"
  await expect(page.locator('.tag.active').first()).toContainText('активный')
})
