import { useState, type FormEvent } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { useT } from '../i18n/LocaleContext'
import { ApiError } from '../api/types'

export default function LoginPage() {
  const { user, login } = useAuth()
  const { t } = useT()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  if (user) return <Navigate to="/polls" replace />

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await login(email, password)
      navigate('/polls')
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.code === 'unauthorized' ? t('login.invalid_credentials') : err.message)
      } else {
        setError(t('login.failed'))
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="auth-page">
      <form className="auth-card" onSubmit={onSubmit}>
        <div className="auth-brand">
          <div className="logo">P</div>
          <h1>{t('login.heading')}</h1>
          <div className="tagline">{t('login.tagline')}</div>
        </div>
        {error && <div className="error">{error}</div>}
        <div className="field">
          <label htmlFor="email">{t('login.email')}</label>
          <input
            id="email"
            type="email"
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </div>
        <div className="field">
          <label htmlFor="password">{t('login.password')}</label>
          <input
            id="password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </div>
        <button type="submit" className="btn primary btn-block" disabled={busy}>
          {busy ? (
            <>
              <span className="spinner" /> {t('login.submitting')}
            </>
          ) : (
            t('login.submit')
          )}
        </button>
        <p className="auth-footer">
          {t('login.no_account')} <Link to="/register">{t('login.create_one')}</Link>
        </p>
      </form>
    </div>
  )
}
