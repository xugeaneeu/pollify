import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import { useT } from '../i18n/LocaleContext'
import { ApiError, type PollDetails, type PollResults } from '../api/types'

export default function PollResultsPage() {
  const { pollId = '' } = useParams<{ pollId: string }>()
  const { t, tn } = useT()
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
      .catch((err) => setError(err instanceof ApiError ? err.message : t('results.failed')))
      .finally(() => setLoading(false))
  }, [pollId, includeVoters, t])

  if (loading)
    return (
      <p className="muted">
        <span className="spinner" /> {t('results.loading')}
      </p>
    )
  if (error) return <div className="error">{error}</div>
  if (!poll || !results) return null

  return (
    <section>
      <Link to={`/polls/${pollId}`} className="back-link">
        {t('results.back')}
      </Link>
      <div className="card">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '1rem' }}>
          <div>
            <h2 style={{ margin: 0 }}>{poll.title}</h2>
            <div style={{ marginTop: '0.5rem' }}>
              <span className={`tag ${results.status}`}>{t(`status.${results.status}`)}</span>
              {poll.is_anonymous && <span className="tag">{t('poll_detail.tag_anonymous')}</span>}
            </div>
          </div>
        </div>

        <div className="stat-row">
          <span>{tn('pluralize.participant', results.participants_count)}</span>
          <span>{tn('pluralize.vote', results.total_votes_count)}</span>
        </div>

        {results.options.length > 0 && (
          <div>
            <h3>{t('results.options_heading')}</h3>
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
            <h3>{t('results.free_text_heading')}</h3>
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
              <label htmlFor="include">{t('results.show_voters')}</label>
            </div>
            {includeVoters && results.voter_details && results.voter_details.length > 0 && (
              <div>
                <h3>{t('results.voters_heading')}</h3>
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
