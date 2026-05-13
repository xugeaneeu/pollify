import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import { useT } from '../i18n/LocaleContext'
import { ApiError, type ReportResponse } from '../api/types'

export default function ReportDetailPage() {
  const { reportId = '' } = useParams<{ reportId: string }>()
  const { t, tn } = useT()
  const [report, setReport] = useState<ReportResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [comment, setComment] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    api
      .getReport(reportId)
      .then(setReport)
      .catch((err) => setError(err instanceof ApiError ? err.message : t('report_detail.failed_load')))
  }, [reportId, t])

  async function review(decision: 'APPROVE' | 'REJECT') {
    setBusy(true)
    setError(null)
    try {
      const outcome = await api.submitReview(reportId, decision, comment || undefined)
      setReport(outcome.report)
      setComment('')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t('report_detail.failed'))
    } finally {
      setBusy(false)
    }
  }

  if (!report && !error)
    return (
      <p className="muted">
        <span className="spinner" /> {t('report_detail.loading')}
      </p>
    )

  return (
    <section>
      <Link to="/moderation" className="back-link">
        {t('report_detail.back')}
      </Link>
      {error && <div className="error">{error}</div>}
      {report && (
        <div className="card">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '1rem' }}>
            <div>
              <h2 style={{ margin: 0 }}>{report.reason}</h2>
              <div className="muted" style={{ marginTop: '0.4rem' }}>
                {t('report_detail.poll_label')}{' '}
                <Link to={`/polls/${report.poll_id}`}>{report.poll_id}</Link> · {t('report_detail.reported_by')}{' '}
                {report.created_by}
              </div>
            </div>
            <span className={`tag ${report.status}`}>{t(`report_status.${report.status}`)}</span>
          </div>
          {report.comment && <p style={{ marginTop: '0.75rem' }}>{report.comment}</p>}
          <div className="stat-row">
            <span>{tn('pluralize.approval', report.approval_count)}</span>
            <span>{tn('pluralize.rejection', report.rejection_count)}</span>
            {report.resolution && (
              <span>
                <strong>{t('report_detail.resolution_label')}</strong> {report.resolution}
              </span>
            )}
          </div>

          <h3>{t('report_detail.review_history')}</h3>
          {report.reviews.length === 0 ? (
            <p className="muted">{t('report_detail.no_reviews')}</p>
          ) : (
            report.reviews.map((rv) => (
              <div key={`${rv.admin_id}-${rv.created_at}`} className="review-item">
                <div className="who">
                  {rv.admin_id}
                  <span className={`tag ${rv.decision === 'APPROVE' ? 'active' : 'hidden'}`}>
                    {rv.decision}
                  </span>
                </div>
                <span className="muted">{new Date(rv.created_at).toLocaleString()}</span>
              </div>
            ))
          )}

          {report.status !== 'RESOLVED' && report.status !== 'REJECTED' && (
            <div style={{ marginTop: '1.5rem', paddingTop: '1.25rem', borderTop: '1px solid var(--border)' }}>
              <h3 style={{ margin: '0 0 0.75rem' }}>{t('report_detail.submit_decision')}</h3>
              <div className="field">
                <label htmlFor="rcomment">{t('report_detail.comment')}</label>
                <textarea
                  id="rcomment"
                  value={comment}
                  onChange={(e) => setComment(e.target.value)}
                  placeholder={t('report_detail.comment_placeholder')}
                />
              </div>
              <div className="toolbar" style={{ marginTop: 0, paddingTop: 0, borderTop: 'none' }}>
                <button className="btn primary" disabled={busy} onClick={() => review('APPROVE')}>
                  {t('decision.APPROVE')}
                </button>
                <button className="btn danger" disabled={busy} onClick={() => review('REJECT')}>
                  {t('decision.REJECT')}
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </section>
  )
}
