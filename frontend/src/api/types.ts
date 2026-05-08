export type UserRole = 'USER' | 'ADMIN'

export interface UserSummary {
  id: string
  display_name: string | null
  email: string | null
  role: UserRole
  created_at: string | null
}

export interface AuthTokens {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
}

export interface AuthSession {
  user: UserSummary
  tokens: AuthTokens
}

export type PollStatus = 'scheduled' | 'active' | 'completed' | 'hidden'

export interface PollOption {
  id: string
  text: string
}

export interface PollSettings {
  is_anonymous: boolean
  is_multiple_choice: boolean
  max_choices: number | null
  allow_custom_answer: boolean
  start_at: string
  end_at: string
}

export interface ParticipationSummary {
  participants_count: number
  has_voted: boolean
}

export interface PollSummary {
  id: string
  title: string
  description: string | null
  question: string
  status: PollStatus
  is_anonymous: boolean
  is_multiple_choice: boolean
  max_choices: number | null
  allow_custom_answer: boolean
  start_at: string
  end_at: string
  is_hidden: boolean
  created_by: string
  created_at: string
  participation_summary: ParticipationSummary
}

export interface PollDetails extends PollSummary {
  options: PollOption[]
  settings: PollSettings
}

export interface PaginationMeta {
  page: number
  limit: number
  total_items: number
  total_pages: number
}

export interface PollsListResponse {
  items: PollSummary[]
  pagination: PaginationMeta
}

export interface CreatePollRequest {
  title: string
  description?: string | null
  question: string
  options?: { text: string }[]
  settings: {
    is_anonymous: boolean
    is_multiple_choice: boolean
    max_choices?: number | null
    allow_custom_answer: boolean
    start_at: string
    end_at: string
  }
}

export interface OptionResult {
  option_id: string
  label: string
  votes_count: number
  percentage: number
}

export interface TextAnswer {
  value: string
  count: number
}

export interface VoterDetail {
  user_id: string
  display_name: string | null
  selected_option_ids: string[]
  custom_text: string | null
}

export interface PollResults {
  poll_id: string
  status: PollStatus
  participants_count: number
  total_votes_count: number
  options: OptionResult[]
  custom_answers: TextAnswer[]
  voter_details?: VoterDetail[]
}

export interface ReportReview {
  admin_id: string
  decision: 'APPROVE' | 'REJECT'
  comment: string | null
  created_at: string
}

export type ReportStatus = 'OPEN' | 'IN_REVIEW' | 'RESOLVED' | 'REJECTED'

export interface ReportResponse {
  id: string
  poll_id: string
  created_by: string
  reason: string
  comment: string | null
  status: ReportStatus
  created_at: string
  approval_count: number
  rejection_count: number
  resolution: string | null
  reviews: ReportReview[]
}

export interface ReportsListResponse {
  items: ReportResponse[]
  pagination: PaginationMeta
}

export interface ReviewOutcome {
  review: ReportReview
  report: ReportResponse
}

export interface ApiErrorBody {
  code: string
  message: string
  details: Record<string, unknown>
}

export class ApiError extends Error {
  status: number
  code: string
  details: Record<string, unknown>

  constructor(status: number, body: ApiErrorBody) {
    super(body.message)
    this.status = status
    this.code = body.code
    this.details = body.details ?? {}
  }
}
