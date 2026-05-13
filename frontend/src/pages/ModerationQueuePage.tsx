import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import { useT } from '../i18n/LocaleContext'
import { ApiError, type ReportResponse, type ReportStatus } from '../api/types'

export default function ModerationQueuePage() {
  const { t, tn } = useT()
  const [reports, setReports] = useState<ReportResponse[]>([])
  const [status, setStatus] = useState<ReportStatus | ''>('OPEN')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const filters = useMemo<Array<{ label: string; value: ReportStatus | '' }>>(
    () => [
      { label: t('moderation.filter_open'), value: 'OPEN' },
      { label: t('moderation.filter_in_review'), value: 'IN_REVIEW' },
      { label: t('moderation.filter_resolved'), value: 'RESOLVED' },
      { label: t('moderation.filter_rejected'), value: 'REJECTED' },
      { label: t('moderation.filter_all'), value: '' },
    ],
    [t],
  )

  useEffect(() => {
    setLoading(true)
    api
      .listReports({ status: status || undefined, limit: 50 })
      .then((res) => setReports(res.items))
      .catch((err) => setError(err instanceof ApiError ? err.message : t('moderation.failed')))
      .finally(() => setLoading(false))
  }, [status, t])

  return (
    <section>
      <div className="section-header">
        <div>
          <h2>{t('moderation.heading')}</h2>
          <div className="subtitle">{t('moderation.subtitle')}</div>
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
          <span className="spinner" /> {t('moderation.loading')}
        </p>
      ) : null}

      {!loading && reports.length === 0 ? (
        <div className="empty-state">
          <div className="emoji">✅</div>
          <p>{t('moderation.empty')}</p>
        </div>
      ) : null}

      {reports.map((r) => (
        <Link key={r.id} to={`/reports/${r.id}`} className="card interactive" style={{ display: 'block' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '1rem' }}>
            <div>
              <strong style={{ fontSize: '1rem' }}>{r.reason}</strong>
              <div className="muted" style={{ marginTop: '0.25rem' }}>
                {t('moderation.poll_label')} <code>{r.poll_id}</code> · {new Date(r.created_at).toLocaleString()}
              </div>
            </div>
            <span className={`tag ${r.status}`}>{t(`report_status.${r.status}`)}</span>
          </div>
          {r.comment && <p style={{ margin: '0.75rem 0 0', color: 'var(--text-muted)' }}>{r.comment}</p>}
          <div className="stat-row" style={{ marginTop: '0.5rem' }}>
            <span>{tn('pluralize.approval', r.approval_count)}</span>
            <span>{tn('pluralize.rejection', r.rejection_count)}</span>
            {r.resolution && (
              <span>
                <strong>{t('moderation.resolution_label')}</strong> {r.resolution}
              </span>
            )}
          </div>
        </Link>
      ))}
    </section>
  )
}
