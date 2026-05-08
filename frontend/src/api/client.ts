import {
  ApiError,
  type AuthSession,
  type CreatePollRequest,
  type PollDetails,
  type PollResults,
  type PollsListResponse,
  type ReportResponse,
  type ReportsListResponse,
  type ReviewOutcome,
  type UserSummary,
} from './types'

const TOKEN_KEY = 'pollify.token'
const USER_KEY = 'pollify.user'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setSession(session: AuthSession): void {
  localStorage.setItem(TOKEN_KEY, session.tokens.access_token)
  localStorage.setItem(USER_KEY, JSON.stringify(session.user))
}

export function clearSession(): void {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
}

export function getStoredUser(): UserSummary | null {
  const raw = localStorage.getItem(USER_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as UserSummary
  } catch {
    return null
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {}
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  const token = getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`

  const res = await fetch(path, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  if (res.status === 204) return undefined as T
  const text = await res.text()
  const data = text ? JSON.parse(text) : null
  if (!res.ok) {
    const errBody = data?.error ?? { code: 'unknown', message: res.statusText, details: {} }
    throw new ApiError(res.status, errBody)
  }
  return data as T
}

export const api = {
  register(email: string, password: string, displayName?: string) {
    return request<AuthSession>('POST', '/api/v1/auth/register', {
      email,
      password,
      display_name: displayName ?? null,
    })
  },
  login(email: string, password: string) {
    return request<AuthSession>('POST', '/api/v1/auth/login', { email, password })
  },
  me() {
    return request<UserSummary>('GET', '/api/v1/users/me')
  },
  listPolls(query: Record<string, string | number | boolean | undefined> = {}) {
    const qs = new URLSearchParams()
    for (const [k, v] of Object.entries(query)) {
      if (v !== undefined && v !== '') qs.set(k, String(v))
    }
    const tail = qs.toString() ? `?${qs.toString()}` : ''
    return request<PollsListResponse>('GET', `/api/v1/polls/${tail}`)
  },
  getPoll(id: string) {
    return request<PollDetails>('GET', `/api/v1/polls/${id}`)
  },
  createPoll(payload: CreatePollRequest) {
    return request<PollDetails>('POST', '/api/v1/polls/', payload)
  },
  vote(pollId: string, payload: { option_ids?: string[]; custom_text?: string }) {
    return request<{ poll_id: string; participation_recorded: boolean }>(
      'POST',
      `/api/v1/polls/${pollId}/votes`,
      payload,
    )
  },
  getResults(pollId: string, includeVoters = false) {
    const qs = includeVoters ? '?include_voters=true' : ''
    return request<PollResults>('GET', `/api/v1/polls/${pollId}/results${qs}`)
  },
  createReport(pollId: string, reason: string, comment?: string) {
    return request<ReportResponse>('POST', `/api/v1/polls/${pollId}/reports`, {
      reason,
      comment: comment ?? null,
    })
  },
  listReports(query: Record<string, string | number | undefined> = {}) {
    const qs = new URLSearchParams()
    for (const [k, v] of Object.entries(query)) {
      if (v !== undefined && v !== '') qs.set(k, String(v))
    }
    const tail = qs.toString() ? `?${qs.toString()}` : ''
    return request<ReportsListResponse>('GET', `/api/v1/reports/${tail}`)
  },
  getReport(id: string) {
    return request<ReportResponse>('GET', `/api/v1/reports/${id}`)
  },
  submitReview(reportId: string, decision: 'APPROVE' | 'REJECT', comment?: string) {
    return request<ReviewOutcome>('POST', `/api/v1/reports/${reportId}/reviews`, {
      decision,
      comment: comment ?? null,
    })
  },
}
