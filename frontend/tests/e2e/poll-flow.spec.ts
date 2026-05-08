import { expect, test } from '@playwright/test'
import { resetDb } from './helpers/db'
import { uniqueEmail } from './helpers/factories'
import { createOptionPoll, createTextPoll, registerUser, signOut } from './helpers/flows'

test.beforeEach(() => {
  resetDb()
})

test('create option poll, vote, see results, double vote rejected', async ({ page }) => {
  const email = uniqueEmail('alice')
  await registerUser(page, email, 'password123', 'Alice')

  const pollId = await createOptionPoll(page, {
    title: 'Lunch?',
    question: 'Pick a place',
    options: ['Pizza', 'Sushi'],
  })

  await expect(page.getByRole('heading', { name: 'Lunch?' })).toBeVisible()
  await expect(page.getByText('active', { exact: true })).toBeVisible()

  await page.getByRole('button', { name: 'Pizza' }).click()
  await page.getByRole('button', { name: 'Submit vote' }).click()

  await expect(page).toHaveURL(new RegExp(`/polls/${pollId}/results$`))
  await expect(page.getByText('1 vote', { exact: true })).toBeVisible()
  await expect(page.getByText('1 participant', { exact: true })).toBeVisible()

  await page.goto(`/polls/${pollId}`)
  await expect(page.getByRole('button', { name: 'Already voted' })).toBeDisabled()
})

test('multi-choice poll respects max_choices', async ({ page }) => {
  await registerUser(page, uniqueEmail('alice'), 'password123', 'Alice')
  const pollId = await createOptionPoll(page, {
    title: 'Toppings',
    question: 'Pick up to two',
    options: ['Cheese', 'Mushrooms', 'Olives'],
    isMultipleChoice: true,
    maxChoices: 2,
  })

  await page.getByRole('button', { name: 'Cheese' }).click()
  await page.getByRole('button', { name: 'Mushrooms' }).click()
  await page.getByRole('button', { name: 'Olives' }).click()

  // Third click should be ignored — only first two stay selected.
  await expect(page.getByRole('button', { name: 'Cheese' })).toHaveClass(/selected/)
  await expect(page.getByRole('button', { name: 'Mushrooms' })).toHaveClass(/selected/)
  await expect(page.getByRole('button', { name: 'Olives' })).not.toHaveClass(/selected/)

  await page.getByRole('button', { name: 'Submit vote' }).click()
  await expect(page).toHaveURL(new RegExp(`/polls/${pollId}/results$`))
  await expect(page.getByText('2 votes', { exact: true })).toBeVisible()
})

test('free-text poll accepts a custom answer', async ({ page }) => {
  await registerUser(page, uniqueEmail('alice'), 'password123', 'Alice')
  const pollId = await createTextPoll(page, { title: 'How are you?', question: 'Describe in one word' })

  await page.getByLabel('Your answer').fill('happy')
  await page.getByRole('button', { name: 'Submit vote' }).click()

  await expect(page).toHaveURL(new RegExp(`/polls/${pollId}/results$`))
  await expect(page.getByRole('heading', { name: 'Free-text answers' })).toBeVisible()
  await expect(page.getByText('happy')).toBeVisible()
})

test('anonymous poll hides voter details from results', async ({ page }) => {
  await registerUser(page, uniqueEmail('alice'), 'password123', 'Alice')
  const pollId = await createOptionPoll(page, {
    title: 'Secret ballot',
    question: 'Choose',
    options: ['A', 'B'],
    isAnonymous: true,
  })

  await page.getByRole('button', { name: 'A' }).click()
  await page.getByRole('button', { name: 'Submit vote' }).click()
  await expect(page).toHaveURL(new RegExp(`/polls/${pollId}/results$`))
  await expect(page.getByText('Show who voted')).toHaveCount(0)
})

test('non-anonymous poll exposes voter on demand', async ({ page }) => {
  await registerUser(page, uniqueEmail('alice'), 'password123', 'Alice')
  const pollId = await createOptionPoll(page, {
    title: 'Open vote',
    question: 'Choose',
    options: ['Yes', 'No'],
  })

  await page.getByRole('button', { name: 'Yes' }).click()
  await page.getByRole('button', { name: 'Submit vote' }).click()
  await expect(page).toHaveURL(new RegExp(`/polls/${pollId}/results$`))

  await page.getByLabel('Show who voted').check()
  await expect(page.getByRole('heading', { name: 'Voters' })).toBeVisible()
  await expect(page.locator('section').getByText('Alice', { exact: true })).toBeVisible()
})

test('list page displays new polls and the "you voted" pill', async ({ page }) => {
  await registerUser(page, uniqueEmail('alice'), 'password123', 'Alice')
  const pollId = await createOptionPoll(page, {
    title: 'Lunch?',
    question: 'Pick',
    options: ['Pizza', 'Sushi'],
  })
  await page.getByRole('button', { name: 'Pizza' }).click()
  await page.getByRole('button', { name: 'Submit vote' }).click()

  await page.goto('/polls')
  await expect(page.getByRole('heading', { name: 'Lunch?' })).toBeVisible()
  await expect(page.getByText('You voted')).toBeVisible()
  await page.getByRole('heading', { name: 'Lunch?' }).click()
  await expect(page).toHaveURL(new RegExp(`/polls/${pollId}$`))
})
