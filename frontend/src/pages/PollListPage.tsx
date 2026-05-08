import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { PollSummary } from '../api/types'
import { ApiError } from '../api/types'

export default function PollListPage() {
  const [polls, setPolls] = useState<PollSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api
      .listPolls({ limit: 50 })
      .then((res) => setPolls(res.items))
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Failed to load polls'))
      .finally(() => setLoading(false))
  }, [])

  return (
    <section>
      <div className="section-header">
        <div>
          <h2>Polls</h2>
          <div className="subtitle">Browse, vote, or start your own.</div>
        </div>
        <Link to="/polls/new" className="btn primary">
          + New poll
        </Link>
      </div>

      {error && <div className="error">{error}</div>}
      {loading ? (
        <p className="muted">
          <span className="spinner" /> Loading polls…
        </p>
      ) : null}

      {!loading && polls.length === 0 ? (
        <div className="empty-state">
          <div className="emoji">📊</div>
          <p>No polls yet.</p>
          <p>
            <Link to="/polls/new">Create the first one</Link> and share it with your team.
          </p>
        </div>
      ) : null}

      <div className="poll-grid">
        {polls.map((p) => (
          <Link key={p.id} to={`/polls/${p.id}`} className="card interactive poll-card">
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '0.5rem' }}>
              <h3>{p.title}</h3>
              <span className={`tag ${p.status}`}>{p.status}</span>
            </div>
            <div style={{ marginTop: '0.35rem' }}>
              {p.is_anonymous && <span className="tag">anon</span>}
              {p.is_multiple_choice && <span className="tag">multi</span>}
              {p.allow_custom_answer && <span className="tag">free text</span>}
            </div>
            <p className="question">{p.question}</p>
            <div className="meta">
              <span>
                {p.participation_summary.participants_count}{' '}
                {p.participation_summary.participants_count === 1 ? 'participant' : 'participants'}
              </span>
              {p.participation_summary.has_voted ? <span className="voted-pill">You voted</span> : null}
            </div>
          </Link>
        ))}
      </div>
    </section>
  )
}
