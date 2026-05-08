import { execFileSync } from 'node:child_process'

const CONTAINER = process.env.POLLIFY_PG_CONTAINER ?? 'pollify-postgres-1'
const USER = process.env.POLLIFY_PG_USER ?? 'pollify'
const DB = process.env.POLLIFY_PG_DB ?? 'pollify'

function psql(sql: string): string {
  return execFileSync(
    'docker',
    ['exec', '-i', CONTAINER, 'psql', '-U', USER, '-d', DB, '-Atc', sql],
    { encoding: 'utf8' },
  ).trim()
}

const TABLES = ['report_reviews', 'reports', 'votes', 'poll_participants', 'options', 'polls', 'users']

export function resetDb(): void {
  psql(`TRUNCATE ${TABLES.join(', ')} CASCADE;`)
}

export function promoteToAdmin(email: string): void {
  const escaped = email.replace(/'/g, "''")
  psql(`UPDATE users SET role='ADMIN' WHERE email='${escaped}';`)
}
