import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import { ApiError, type ReportResponse } from '../api/types'

export default function ReportDetailPage() {
  const { reportId = '' } = useParams<{ reportId: string }>()
  const [report, setReport] = useState<ReportResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [comment, setComment] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    api
      .getReport(reportId)
      .then(setReport)
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Failed to load report'))
  }, [reportId])

  async function review(decision: 'APPROVE' | 'REJECT') {
    setBusy(true)
    setError(null)
    try {
      const outcome = await api.submitReview(reportId, decision, comment || undefined)
      setReport(outcome.report)
      setComment('')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to submit review')
    } finally {
      setBusy(false)
    }
  }

  if (!report && !error)
    return (
      <p className="muted">
        <span className="spinner" /> Loading report…
      </p>
    )

  return (
    <section>
      <Link to="/moderation" className="back-link">
        ← Back to queue
      </Link>
      {error && <div className="error">{error}</div>}
      {report && (
        <div className="card">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '1rem' }}>
            <div>
              <h2 style={{ margin: 0 }}>{report.reason}</h2>
              <div className="muted" style={{ marginTop: '0.4rem' }}>
                Poll <Link to={`/polls/${report.poll_id}`}>{report.poll_id}</Link> · reported by{' '}
                {report.created_by}
              </div>
            </div>
            <span className={`tag ${report.status}`}>{report.status.replace('_', ' ')}</span>
          </div>
          {report.comment && <p style={{ marginTop: '0.75rem' }}>{report.comment}</p>}
          <div className="stat-row">
            <span>
              <strong>{report.approval_count}</strong> approvals
            </span>
            <span>
              <strong>{report.rejection_count}</strong> rejections
            </span>
            {report.resolution && (
              <span>
                <strong>Resolution:</strong> {report.resolution}
              </span>
            )}
          </div>

          <h3>Review history</h3>
          {report.reviews.length === 0 ? (
            <p className="muted">No reviews yet.</p>
          ) : (
            report.reviews.map((rv) => (
              <div key={`${rv.admin_id}-${rv.created_at}`} className="review-item">
                <div className="who">
                  {rv.admin_id}
                  <span className={`tag ${rv.decision === 'APPROVE' ? 'active' : 'hidden'}`}>{rv.decision}</span>
                </div>
                <span className="muted">{new Date(rv.created_at).toLocaleString()}</span>
              </div>
            ))
          )}

          {report.status !== 'RESOLVED' && report.status !== 'REJECTED' && (
            <div style={{ marginTop: '1.5rem', paddingTop: '1.25rem', borderTop: '1px solid var(--border)' }}>
              <h3 style={{ margin: '0 0 0.75rem' }}>Submit your decision</h3>
              <div className="field">
                <label htmlFor="rcomment">Comment (optional)</label>
                <textarea
                  id="rcomment"
                  value={comment}
                  onChange={(e) => setComment(e.target.value)}
                  placeholder="Why are you approving or rejecting?"
                />
              </div>
              <div className="toolbar" style={{ marginTop: 0, paddingTop: 0, borderTop: 'none' }}>
                <button className="btn primary" disabled={busy} onClick={() => review('APPROVE')}>
                  Approve
                </button>
                <button className="btn danger" disabled={busy} onClick={() => review('REJECT')}>
                  Reject
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </section>
  )
}
