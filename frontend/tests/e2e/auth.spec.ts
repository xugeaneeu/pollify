import { expect, test } from '@playwright/test'
import { resetDb } from './helpers/db'
import { uniqueEmail } from './helpers/factories'
import { loginAs, registerUser, signOut } from './helpers/flows'

test.beforeEach(() => {
  resetDb()
})

test('register, navigate to polls, sign out, sign back in', async ({ page }) => {
  const email = uniqueEmail('alice')
  await registerUser(page, email, 'password123', 'Alice')
  await expect(page.getByRole('heading', { name: 'Polls' })).toBeVisible()
  await expect(page.getByText('Alice')).toBeVisible()

  await signOut(page)
  await loginAs(page, email, 'password123')
  await expect(page.getByRole('heading', { name: 'Polls' })).toBeVisible()
})

test('shows error for invalid login credentials', async ({ page }) => {
  await page.goto('/login')
  await page.getByLabel('Email').fill('nobody@example.com')
  await page.getByLabel('Password').fill('wrongpass')
  await page.getByRole('button', { name: /Sign in/i }).click()
  await expect(page.getByText(/invalid credentials/i)).toBeVisible()
  await expect(page).toHaveURL(/\/login$/)
})

test('redirects unauthenticated user to login', async ({ page }) => {
  await page.goto('/polls')
  await expect(page).toHaveURL(/\/login$/)
})

test('non-admin user does not see Moderation link and is bounced from /moderation', async ({ page }) => {
  const email = uniqueEmail('bob')
  await registerUser(page, email, 'password123', 'Bob')
  await expect(page.getByRole('link', { name: 'Moderation' })).toHaveCount(0)
  await page.goto('/moderation')
  await expect(page).toHaveURL(/\/polls$/)
})
