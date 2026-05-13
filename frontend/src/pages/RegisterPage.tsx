import { useState, type FormEvent } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { useT } from '../i18n/LocaleContext'
import { ApiError } from '../api/types'

export default function RegisterPage() {
  const { user, register } = useAuth()
  const { t } = useT()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  if (user) return <Navigate to="/polls" replace />

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await register(email, password, displayName || undefined)
      navigate('/polls')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t('register.failed'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="auth-page">
      <form className="auth-card" onSubmit={onSubmit}>
        <div className="auth-brand">
          <div className="logo">P</div>
          <h1>{t('register.heading')}</h1>
          <div className="tagline">{t('register.tagline')}</div>
        </div>
        {error && <div className="error">{error}</div>}
        <div className="field">
          <label htmlFor="email">{t('register.email')}</label>
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
          <label htmlFor="display">{t('register.display_name')}</label>
          <input
            id="display"
            type="text"
            autoComplete="nickname"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            placeholder={t('register.display_name_hint')}
          />
        </div>
        <div className="field">
          <label htmlFor="password">{t('register.password')}</label>
          <input
            id="password"
            type="password"
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            minLength={8}
          />
          <div className="help">{t('register.password_hint')}</div>
        </div>
        <button type="submit" className="btn primary btn-block" disabled={busy}>
          {busy ? (
            <>
              <span className="spinner" /> {t('register.submitting')}
            </>
          ) : (
            t('register.submit')
          )}
        </button>
        <p className="auth-footer">
          {t('register.already_have')} <Link to="/login">{t('register.sign_in')}</Link>
        </p>
      </form>
    </div>
  )
}
