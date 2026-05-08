import { expect, test } from '@playwright/test'
import { promoteToAdmin, resetDb } from './helpers/db'
import { uniqueEmail } from './helpers/factories'
import { createOptionPoll, loginAs, registerUser, signOut } from './helpers/flows'

test.beforeEach(() => {
  resetDb()
})

test('user reports poll, admin quorum hides it', async ({ page }) => {
  const voterEmail = uniqueEmail('voter')
  await registerUser(page, voterEmail, 'password123', 'Voter')
  const pollId = await createOptionPoll(page, {
    title: 'Trash poll',
    question: 'Pick',
    options: ['A', 'B'],
  })
  await signOut(page)

  // Reporter
  const reporterEmail = uniqueEmail('reporter')
  await registerUser(page, reporterEmail, 'password123', 'Reporter')
  await page.goto(`/polls/${pollId}`)
  await page.getByRole('button', { name: 'Report poll' }).click()
  await page.getByLabel('Reason').fill('inappropriate')
  await page.getByRole('button', { name: 'Send report' }).click()
  await expect(page.getByText(/Report submitted/i)).toBeVisible()
  await signOut(page)

  // Two admins
  const admin1 = uniqueEmail('admin1')
  const admin2 = uniqueEmail('admin2')
  await registerUser(page, admin1, 'password123', 'Admin1')
  await signOut(page)
  await registerUser(page, admin2, 'password123', 'Admin2')
  await signOut(page)
  promoteToAdmin(admin1)
  promoteToAdmin(admin2)

  // Admin1 approves -> IN_REVIEW
  await loginAs(page, admin1, 'password123')
  await expect(page.getByRole('link', { name: 'Moderation' })).toBeVisible()
  await page.getByRole('link', { name: 'Moderation' }).click()
  await expect(page).toHaveURL(/\/moderation$/)
  await page.getByText('inappropriate').click()
  await page.getByRole('button', { name: 'Approve', exact: true }).click()
  await expect(page.getByText('in review', { exact: false }).first()).toBeVisible()
  await signOut(page)

  // Admin2 approves -> RESOLVED + poll hidden
  await loginAs(page, admin2, 'password123')
  await page.goto('/moderation')
  await page.getByRole('button', { name: 'In review' }).click()
  await page.getByText('inappropriate').click()
  await page.getByRole('button', { name: 'Approve', exact: true }).click()
  await expect(page.locator('.tag.RESOLVED').first()).toBeVisible()
  await expect(page.getByText(/poll_hidden/i)).toBeVisible()
  await signOut(page)

  // Voter sees poll hidden
  await loginAs(page, voterEmail, 'password123')
  await page.goto(`/polls/${pollId}`)
  await expect(page.locator('.tag.hidden').first()).toBeVisible()
})

test('admin cannot review their own report (self-review forbidden)', async ({ page }) => {
  const voterEmail = uniqueEmail('voter')
  await registerUser(page, voterEmail, 'password123', 'Voter')
  const pollId = await createOptionPoll(page, {
    title: 'A poll',
    question: 'Q',
    options: ['A', 'B'],
  })
  await signOut(page)

  // Admin both files report and tries to review
  const adminEmail = uniqueEmail('admin')
  await registerUser(page, adminEmail, 'password123', 'AdminSelf')
  await page.goto(`/polls/${pollId}`)
  await page.getByRole('button', { name: 'Report poll' }).click()
  await page.getByLabel('Reason').fill('rude')
  await page.getByRole('button', { name: 'Send report' }).click()
  await signOut(page)
  promoteToAdmin(adminEmail)

  await loginAs(page, adminEmail, 'password123')
  await page.goto('/moderation')
  await page.getByText('rude').click()
  await page.getByRole('button', { name: 'Approve', exact: true }).click()
  await expect(page.getByText(/admin cannot review own report/i)).toBeVisible()
})

test('admin cannot review the same report twice', async ({ page }) => {
  const voterEmail = uniqueEmail('voter')
  await registerUser(page, voterEmail, 'password123', 'Voter')
  const pollId = await createOptionPoll(page, {
    title: 'Another poll',
    question: 'Q',
    options: ['Yes', 'No'],
  })
  await signOut(page)

  const reporterEmail = uniqueEmail('reporter')
  await registerUser(page, reporterEmail, 'password123', 'Reporter')
  await page.goto(`/polls/${pollId}`)
  await page.getByRole('button', { name: 'Report poll' }).click()
  await page.getByLabel('Reason').fill('off-topic')
  await page.getByRole('button', { name: 'Send report' }).click()
  await signOut(page)

  const adminEmail = uniqueEmail('admin')
  await registerUser(page, adminEmail, 'password123', 'Admin')
  await signOut(page)
  promoteToAdmin(adminEmail)

  await loginAs(page, adminEmail, 'password123')
  await page.goto('/moderation')
  await page.getByText('off-topic').click()
  await page.getByRole('button', { name: 'Approve', exact: true }).click()
  // First click succeeds; the report stays IN_REVIEW. Try to approve again.
  await page.getByRole('button', { name: 'Approve', exact: true }).click()
  await expect(page.getByText(/review already submitted/i)).toBeVisible()
})

test('reject quorum closes report without hiding poll', async ({ page }) => {
  const voterEmail = uniqueEmail('voter')
  await registerUser(page, voterEmail, 'password123', 'Voter')
  const pollId = await createOptionPoll(page, {
    title: 'Innocent poll',
    question: 'Q',
    options: ['A', 'B'],
  })
  await signOut(page)

  const reporterEmail = uniqueEmail('reporter')
  await registerUser(page, reporterEmail, 'password123', 'Reporter')
  await page.goto(`/polls/${pollId}`)
  await page.getByRole('button', { name: 'Report poll' }).click()
  await page.getByLabel('Reason').fill('frivolous')
  await page.getByRole('button', { name: 'Send report' }).click()
  await signOut(page)

  const admin1 = uniqueEmail('admin1')
  const admin2 = uniqueEmail('admin2')
  await registerUser(page, admin1, 'password123', 'Admin1')
  await signOut(page)
  await registerUser(page, admin2, 'password123', 'Admin2')
  await signOut(page)
  promoteToAdmin(admin1)
  promoteToAdmin(admin2)

  await loginAs(page, admin1, 'password123')
  await page.goto('/moderation')
  await page.getByText('frivolous').click()
  await page.getByRole('button', { name: 'Reject', exact: true }).click()
  await signOut(page)

  await loginAs(page, admin2, 'password123')
  await page.goto('/moderation')
  await page.getByRole('button', { name: 'In review' }).click()
  await page.getByText('frivolous').click()
  await page.getByRole('button', { name: 'Reject', exact: true }).click()
  await expect(page.locator('.tag.REJECTED').first()).toBeVisible()
  await signOut(page)

  // Poll is NOT hidden
  await loginAs(page, voterEmail, 'password123')
  await page.goto(`/polls/${pollId}`)
  await expect(page.locator('.tag.active').first()).toBeVisible()
})
