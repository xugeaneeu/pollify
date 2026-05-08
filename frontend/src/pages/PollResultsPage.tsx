import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import { ApiError, type PollDetails, type PollResults } from '../api/types'

export default function PollResultsPage() {
  const { pollId = '' } = useParams<{ pollId: string }>()
  const [poll, setPoll] = useState<PollDetails | null>(null)
  const [results, setResults] = useState<PollResults | null>(null)
  const [includeVoters, setIncludeVoters] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    Promise.all([api.getPoll(pollId), api.getResults(pollId, includeVoters)])
      .then(([pollData, resultsData]) => {
        setPoll(pollData)
        setResults(resultsData)
      })
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Failed to load results'))
      .finally(() => setLoading(false))
  }, [pollId, includeVoters])

  if (loading)
    return (
      <p className="muted">
        <span className="spinner" /> Loading results…
      </p>
    )
  if (error) return <div className="error">{error}</div>
  if (!poll || !results) return null

  return (
    <section>
      <Link to={`/polls/${pollId}`} className="back-link">
        ← Back to poll
      </Link>
      <div className="card">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '1rem' }}>
          <div>
            <h2 style={{ margin: 0 }}>{poll.title}</h2>
            <div style={{ marginTop: '0.5rem' }}>
              <span className={`tag ${results.status}`}>{results.status}</span>
              {poll.is_anonymous && <span className="tag">anonymous</span>}
            </div>
          </div>
        </div>

        <div className="stat-row">
          <span>
            <strong>{results.participants_count}</strong>{' '}
            {results.participants_count === 1 ? 'participant' : 'participants'}
          </span>
          <span>
            <strong>{results.total_votes_count}</strong>{' '}
            {results.total_votes_count === 1 ? 'vote' : 'votes'}
          </span>
        </div>

        {results.options.length > 0 && (
          <div>
            <h3>Options</h3>
            {results.options.map((opt) => (
              <div key={opt.option_id} className="option-row">
                <div style={{ flex: 1 }}>
                  <div style={{ fontWeight: 500 }}>{opt.label}</div>
                  <div className="bar">
                    <span style={{ width: `${opt.percentage}%` }} />
                  </div>
                </div>
                <div style={{ minWidth: 120, textAlign: 'right', fontSize: '0.9rem' }}>
                  <strong>{opt.votes_count}</strong>
                  <span className="muted"> · {opt.percentage.toFixed(1)}%</span>
                </div>
              </div>
            ))}
          </div>
        )}

        {results.custom_answers.length > 0 && (
          <div>
            <h3>Free-text answers</h3>
            {results.custom_answers.map((a) => (
              <div key={a.value} className="option-row">
                <span>{a.value}</span>
                <span className="muted">×{a.count}</span>
              </div>
            ))}
          </div>
        )}

        {!poll.is_anonymous && (
          <div style={{ marginTop: '1.5rem', paddingTop: '1.25rem', borderTop: '1px solid var(--border)' }}>
            <div className="checkbox-row">
              <input
                id="include"
                type="checkbox"
                checked={includeVoters}
                onChange={(e) => setIncludeVoters(e.target.checked)}
              />
              <label htmlFor="include">Show who voted</label>
            </div>
            {includeVoters && results.voter_details && results.voter_details.length > 0 && (
              <div>
                <h3>Voters</h3>
                {results.voter_details.map((v) => (
                  <div key={v.user_id} className="option-row">
                    <span style={{ fontWeight: 500 }}>{v.display_name || v.user_id}</span>
                    <span className="muted">
                      {v.custom_text
                        ? v.custom_text
                        : v.selected_option_ids
                            .map((id) => poll.options.find((o) => o.id === id)?.text ?? id)
                            .join(', ')}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </section>
  )
}
