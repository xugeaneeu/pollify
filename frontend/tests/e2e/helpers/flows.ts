import { expect, type Page } from '@playwright/test'
import { futureLocal, pastLocal } from './factories'

export async function registerUser(
  page: Page,
  email: string,
  password = 'password123',
  displayName?: string,
): Promise<void> {
  await page.goto('/register')
  await page.getByLabel('Email').fill(email)
  if (displayName !== undefined) {
    await page.getByLabel('Display name').fill(displayName)
  }
  await page.getByLabel('Password').fill(password)
  await page.getByRole('button', { name: /Sign up/i }).click()
  await expect(page).toHaveURL(/\/polls$/)
}

export async function loginAs(page: Page, email: string, password = 'password123'): Promise<void> {
  await page.goto('/login')
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Password').fill(password)
  await page.getByRole('button', { name: /Sign in/i }).click()
  await expect(page).toHaveURL(/\/polls$/)
}

export async function signOut(page: Page): Promise<void> {
  await page.getByRole('button', { name: 'Sign out' }).click()
  await expect(page).toHaveURL(/\/login$/)
}

interface CreateOptionPollOpts {
  title: string
  question: string
  options: string[]
  description?: string
  isAnonymous?: boolean
  isMultipleChoice?: boolean
  maxChoices?: number
}

export async function createOptionPoll(page: Page, opts: CreateOptionPollOpts): Promise<string> {
  await page.goto('/polls/new')
  await page.getByLabel('Title').fill(opts.title)
  if (opts.description) await page.getByLabel('Description (optional)').fill(opts.description)
  await page.getByLabel('Question').fill(opts.question)

  if (opts.isAnonymous) await page.getByLabel('Anonymous voting').check()
  if (opts.isMultipleChoice) {
    await page.getByLabel('Multiple choice').check()
    if (opts.maxChoices !== undefined) {
      await page.getByLabel(/Max choices/).fill(String(opts.maxChoices))
    }
  }

  await page.getByLabel('Options (one per line)').fill(opts.options.join('\n'))
  await page.getByLabel('Starts').fill(pastLocal(60))
  await page.getByLabel('Ends').fill(futureLocal(60 * 24))

  await page.getByRole('button', { name: /Create poll/i }).click()
  await expect(page).toHaveURL(/\/polls\/[0-9a-f-]+$/)
  const url = new URL(page.url())
  const id = url.pathname.split('/').pop()!
  return id
}

interface CreateTextPollOpts {
  title: string
  question: string
  description?: string
  isAnonymous?: boolean
}

export async function createTextPoll(page: Page, opts: CreateTextPollOpts): Promise<string> {
  await page.goto('/polls/new')
  await page.getByLabel('Title').fill(opts.title)
  if (opts.description) await page.getByLabel('Description (optional)').fill(opts.description)
  await page.getByLabel('Question').fill(opts.question)
  await page.getByLabel('Free-text answers (no fixed options)').check()
  if (opts.isAnonymous) await page.getByLabel('Anonymous voting').check()
  await page.getByLabel('Starts').fill(pastLocal(60))
  await page.getByLabel('Ends').fill(futureLocal(60 * 24))
  await page.getByRole('button', { name: /Create poll/i }).click()
  await expect(page).toHaveURL(/\/polls\/[0-9a-f-]+$/)
  const url = new URL(page.url())
  return url.pathname.split('/').pop()!
}
