import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import { ApiError, type ReportResponse, type ReportStatus } from '../api/types'

const filters: Array<{ label: string; value: ReportStatus | '' }> = [
  { label: 'Open', value: 'OPEN' },
  { label: 'In review', value: 'IN_REVIEW' },
  { label: 'Resolved', value: 'RESOLVED' },
  { label: 'Rejected', value: 'REJECTED' },
  { label: 'All', value: '' },
]

export default function ModerationQueuePage() {
  const [reports, setReports] = useState<ReportResponse[]>([])
  const [status, setStatus] = useState<ReportStatus | ''>('OPEN')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setLoading(true)
    api
      .listReports({ status: status || undefined, limit: 50 })
      .then((res) => setReports(res.items))
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Failed to load reports'))
      .finally(() => setLoading(false))
  }, [status])

  return (
    <section>
      <div className="section-header">
        <div>
          <h2>Moderation queue</h2>
          <div className="subtitle">Reports awaiting admin review.</div>
        </div>
        <div className="chip-filter">
          {filters.map((f) => (
            <button
              key={f.value || 'ALL'}
              className={status === f.value ? 'active' : ''}
              onClick={() => setStatus(f.value)}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      {error && <div className="error">{error}</div>}
      {loading ? (
        <p className="muted">
          <span className="spinner" /> Loading reports…
        </p>
      ) : null}

      {!loading && reports.length === 0 ? (
        <div className="empty-state">
          <div className="emoji">✅</div>
          <p>No reports match this filter.</p>
        </div>
      ) : null}

      {reports.map((r) => (
        <Link key={r.id} to={`/reports/${r.id}`} className="card interactive" style={{ display: 'block' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '1rem' }}>
            <div>
              <strong style={{ fontSize: '1rem' }}>{r.reason}</strong>
              <div className="muted" style={{ marginTop: '0.25rem' }}>
                Poll <code>{r.poll_id}</code> · {new Date(r.created_at).toLocaleString()}
              </div>
            </div>
            <span className={`tag ${r.status}`}>{r.status.replace('_', ' ')}</span>
          </div>
          {r.comment && <p style={{ margin: '0.75rem 0 0', color: 'var(--text-muted)' }}>{r.comment}</p>}
          <div className="stat-row" style={{ marginTop: '0.5rem' }}>
            <span>
              <strong>{r.approval_count}</strong> approvals
            </span>
            <span>
              <strong>{r.rejection_count}</strong> rejections
            </span>
            {r.resolution && (
              <span>
                <strong>Resolution:</strong> {r.resolution}
              </span>
            )}
          </div>
        </Link>
      ))}
    </section>
  )
}
