import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import { useAuth } from '../auth/AuthContext'
import { useT } from '../i18n/LocaleContext'
import { ApiError, type PollStatus, type PollSummary } from '../api/types'

export default function PollListPage() {
  const { user } = useAuth()
  const { t, tn } = useT()
  const [polls, setPolls] = useState<PollSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [status, setStatus] = useState<PollStatus | ''>('')
  const [availableOnly, setAvailableOnly] = useState(false)
  const [anonymousOnly, setAnonymousOnly] = useState(false)
  const [mineOnly, setMineOnly] = useState(false)

  const statusFilters = useMemo<Array<{ label: string; value: PollStatus | '' }>>(
    () => [
      { label: t('poll_list.filter_all'), value: '' },
      { label: t('poll_list.filter_active'), value: 'active' },
      { label: t('poll_list.filter_scheduled'), value: 'scheduled' },
      { label: t('poll_list.filter_completed'), value: 'completed' },
    ],
    [t],
  )

  useEffect(() => {
    setLoading(true)
    setError(null)
    api
      .listPolls({
        status: status || undefined,
        available_for_voting: availableOnly || undefined,
        is_anonymous: anonymousOnly || undefined,
        creator_id: mineOnly && user ? user.id : undefined,
        limit: 50,
      })
      .then((res) => setPolls(res.items))
      .catch((err) => setError(err instanceof ApiError ? err.message : t('poll_list.failed')))
      .finally(() => setLoading(false))
  }, [status, availableOnly, anonymousOnly, mineOnly, user?.id, t])

  const activeFilters =
    (status ? 1 : 0) + (availableOnly ? 1 : 0) + (anonymousOnly ? 1 : 0) + (mineOnly ? 1 : 0)

  function clearFilters() {
    setStatus('')
    setAvailableOnly(false)
    setAnonymousOnly(false)
    setMineOnly(false)
  }

  return (
    <section>
      <div className="section-header">
        <div>
          <h2>{t('poll_list.heading')}</h2>
          <div className="subtitle">{t('poll_list.subtitle')}</div>
        </div>
        <Link to="/polls/new" className="btn primary">
          {t('poll_list.new_poll')}
        </Link>
      </div>

      <div className="filter-bar">
        <div className="chip-filter">
          {statusFilters.map((f) => (
            <button
              key={f.value || 'ALL'}
              className={status === f.value ? 'active' : ''}
              onClick={() => setStatus(f.value)}
            >
              {f.label}
            </button>
          ))}
        </div>
        <label className="toggle">
          <input
            type="checkbox"
            checked={availableOnly}
            onChange={(e) => setAvailableOnly(e.target.checked)}
          />
          {t('poll_list.toggle_available')}
        </label>
        <label className="toggle">
          <input
            type="checkbox"
            checked={anonymousOnly}
            onChange={(e) => setAnonymousOnly(e.target.checked)}
          />
          {t('poll_list.toggle_anonymous')}
        </label>
        <label className="toggle">
          <input
            type="checkbox"
            checked={mineOnly}
            onChange={(e) => setMineOnly(e.target.checked)}
            disabled={!user}
          />
          {t('poll_list.toggle_mine')}
        </label>
        {activeFilters > 0 && (
          <button type="button" className="btn ghost filter-clear" onClick={clearFilters}>
            {t('poll_list.clear_n', { count: activeFilters })}
          </button>
        )}
      </div>

      {error && <div className="error">{error}</div>}
      {loading ? (
        <p className="muted">
          <span className="spinner" /> {t('poll_list.loading')}
        </p>
      ) : null}

      {!loading && polls.length === 0 ? (
        <div className="empty-state">
          <div className="emoji">📊</div>
          {activeFilters > 0 ? (
            <>
              <p>{t('poll_list.empty_filter')}</p>
              <p>
                <button type="button" className="btn ghost" onClick={clearFilters}>
                  {t('poll_list.empty_filter_action')}
                </button>
              </p>
            </>
          ) : (
            <>
              <p>{t('poll_list.empty_none_1')}</p>
              <p>
                <Link to="/polls/new">{t('poll_list.empty_none_2')}</Link> {t('poll_list.empty_none_3')}
              </p>
            </>
          )}
        </div>
      ) : null}

      <div className="poll-grid">
        {polls.map((p) => (
          <Link key={p.id} to={`/polls/${p.id}`} className="card interactive poll-card">
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '0.5rem' }}>
              <h3>{p.title}</h3>
              <span className={`tag ${p.status}`}>{t(`status.${p.status}`)}</span>
            </div>
            <div style={{ marginTop: '0.35rem' }}>
              {p.is_anonymous && <span className="tag">{t('poll_list.tag_anon')}</span>}
              {p.is_multiple_choice && <span className="tag">{t('poll_list.tag_multi')}</span>}
              {p.allow_custom_answer && <span className="tag">{t('poll_list.tag_free')}</span>}
            </div>
            <p className="question">{p.question}</p>
            <div className="meta">
              <span>{tn('pluralize.participant', p.participation_summary.participants_count)}</span>
              {p.participation_summary.has_voted ? (
                <span className="voted-pill">{t('poll_list.you_voted')}</span>
              ) : null}
            </div>
          </Link>
        ))}
      </div>
    </section>
  )
}
