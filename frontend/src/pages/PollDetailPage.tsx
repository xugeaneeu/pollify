import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { api } from '../api/client'
import { ApiError, type PollDetails } from '../api/types'

export default function PollDetailPage() {
  const { pollId = '' } = useParams<{ pollId: string }>()
  const navigate = useNavigate()
  const [poll, setPoll] = useState<PollDetails | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const [selectedOptions, setSelectedOptions] = useState<string[]>([])
  const [customText, setCustomText] = useState('')

  const [reportOpen, setReportOpen] = useState(false)
  const [reportReason, setReportReason] = useState('')
  const [reportComment, setReportComment] = useState('')
  const [reportNotice, setReportNotice] = useState<string | null>(null)
  const [reportError, setReportError] = useState<string | null>(null)

  useEffect(() => {
    setLoading(true)
    api
      .getPoll(pollId)
      .then(setPoll)
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Failed to load poll'))
      .finally(() => setLoading(false))
  }, [pollId])

  function toggleOption(id: string) {
    if (!poll) return
    if (poll.is_multiple_choice) {
      setSelectedOptions((prev) => {
        const next = prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]
        const limit = poll.max_choices ?? next.length
        return next.slice(0, limit)
      })
    } else {
      setSelectedOptions([id])
    }
  }

  async function submitVote() {
    if (!poll) return
    setBusy(true)
    setError(null)
    try {
      const payload: { option_ids?: string[]; custom_text?: string } = poll.allow_custom_answer
        ? { custom_text: customText }
        : { option_ids: selectedOptions }
      await api.vote(poll.id, payload)
      navigate(`/polls/${poll.id}/results`)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to submit vote')
    } finally {
      setBusy(false)
    }
  }

  async function submitReport() {
    if (!poll) return
    setReportError(null)
    setReportNotice(null)
    try {
      await api.createReport(poll.id, reportReason, reportComment || undefined)
      setReportNotice('Report submitted. An admin will review it.')
      setReportReason('')
      setReportComment('')
      setReportOpen(false)
    } catch (err) {
      setReportError(err instanceof ApiError ? err.message : 'Failed to submit report')
    }
  }

  if (loading)
    return (
      <p className="muted">
        <span className="spinner" /> Loading poll…
      </p>
    )
  if (error && !poll) return <div className="error">{error}</div>
  if (!poll) return null

  const canVote = poll.status === 'active' && !poll.participation_summary.has_voted
  const submitDisabled =
    busy ||
    !canVote ||
    (poll.allow_custom_answer ? customText.trim().length === 0 : selectedOptions.length === 0)

  return (
    <section>
      <Link to="/polls" className="back-link">
        ← All polls
      </Link>
      <div className="card">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '1rem' }}>
          <div>
            <h2 style={{ margin: 0 }}>{poll.title}</h2>
            <div style={{ marginTop: '0.5rem' }}>
              <span className={`tag ${poll.status}`}>{poll.status}</span>
              {poll.is_anonymous && <span className="tag">anonymous</span>}
              {poll.is_multiple_choice && <span className="tag">multi · max {poll.max_choices}</span>}
              {poll.allow_custom_answer && <span className="tag">free text</span>}
            </div>
          </div>
        </div>

        {poll.description && <p style={{ color: 'var(--text-muted)' }}>{poll.description}</p>}

        <p style={{ fontSize: '1.05rem', marginTop: '1rem' }}>
          <strong>{poll.question}</strong>
        </p>

        <div className="stat-row">
          <span>
            <strong>{poll.participation_summary.participants_count}</strong>{' '}
            {poll.participation_summary.participants_count === 1 ? 'participant' : 'participants'}
          </span>
          <span>
            <strong>Starts</strong> {new Date(poll.start_at).toLocaleString()}
          </span>
          <span>
            <strong>Ends</strong> {new Date(poll.end_at).toLocaleString()}
          </span>
        </div>

        {error && (
          <div className="error" style={{ marginTop: '1rem' }}>
            {error}
          </div>
        )}
        {reportNotice && (
          <div className="notice" style={{ marginTop: '1rem' }}>
            {reportNotice}
          </div>
        )}

        <h3>Cast your vote</h3>

        {poll.allow_custom_answer ? (
          <div className="field">
            <label htmlFor="answer">Your answer</label>
            <textarea
              id="answer"
              value={customText}
              onChange={(e) => setCustomText(e.target.value)}
              disabled={!canVote}
              placeholder={canVote ? 'Type your answer…' : 'Voting unavailable'}
            />
          </div>
        ) : (
          <div>
            {poll.options.map((opt) => {
              const selected = selectedOptions.includes(opt.id)
              return (
                <button
                  key={opt.id}
                  type="button"
                  className={`option-row ${selected ? 'selected' : ''}`}
                  disabled={!canVote}
                  onClick={() => toggleOption(opt.id)}
                >
                  <span>{opt.text}</span>
                  {selected ? <span className="tag active">selected</span> : null}
                </button>
              )
            })}
          </div>
        )}

        <div className="toolbar">
          <button className="btn primary" onClick={submitVote} disabled={submitDisabled}>
            {poll.participation_summary.has_voted
              ? 'Already voted'
              : busy
              ? 'Submitting…'
              : 'Submit vote'}
          </button>
          <Link className="btn" to={`/polls/${poll.id}/results`}>
            View results
          </Link>
          <button className="btn ghost" onClick={() => setReportOpen((v) => !v)}>
            {reportOpen ? 'Cancel report' : 'Report poll'}
          </button>
        </div>

        {reportOpen && (
          <div style={{ marginTop: '1.25rem', borderTop: '1px solid var(--border)', paddingTop: '1.25rem' }}>
            <h3 style={{ margin: '0 0 0.75rem' }}>Submit a report</h3>
            {reportError && <div className="error">{reportError}</div>}
            <div className="field">
              <label htmlFor="reason">Reason</label>
              <input
                id="reason"
                value={reportReason}
                onChange={(e) => setReportReason(e.target.value)}
                placeholder="What's wrong with this poll?"
                required
              />
            </div>
            <div className="field">
              <label htmlFor="comment">Comment (optional)</label>
              <textarea
                id="comment"
                value={reportComment}
                onChange={(e) => setReportComment(e.target.value)}
                placeholder="Anything else admins should know?"
              />
            </div>
            <button className="btn danger" onClick={submitReport} disabled={!reportReason.trim()}>
              Send report
            </button>
          </div>
        )}
      </div>
    </section>
  )
}
