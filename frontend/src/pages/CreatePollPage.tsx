import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import { useT } from '../i18n/LocaleContext'
import { ApiError } from '../api/types'

function defaultStart() {
  const now = new Date()
  now.setMinutes(now.getMinutes() - now.getTimezoneOffset())
  return now.toISOString().slice(0, 16)
}

function defaultEnd() {
  const now = new Date()
  now.setDate(now.getDate() + 1)
  now.setMinutes(now.getMinutes() - now.getTimezoneOffset())
  return now.toISOString().slice(0, 16)
}

export default function CreatePollPage() {
  const { t } = useT()
  const navigate = useNavigate()
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [question, setQuestion] = useState('')
  const [optionsText, setOptionsText] = useState('')
  const [isAnonymous, setIsAnonymous] = useState(false)
  const [isMultiple, setIsMultiple] = useState(false)
  const [maxChoices, setMaxChoices] = useState(2)
  const [allowCustom, setAllowCustom] = useState(false)
  const [startAt, setStartAt] = useState(defaultStart())
  const [endAt, setEndAt] = useState(defaultEnd())
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      const options = optionsText
        .split('\n')
        .map((line) => line.trim())
        .filter(Boolean)
        .map((text) => ({ text }))
      const payload = {
        title,
        description: description || null,
        question,
        options: allowCustom ? [] : options,
        settings: {
          is_anonymous: isAnonymous,
          is_multiple_choice: isMultiple,
          max_choices: isMultiple ? maxChoices : null,
          allow_custom_answer: allowCustom,
          start_at: new Date(startAt).toISOString(),
          end_at: new Date(endAt).toISOString(),
        },
      }
      const created = await api.createPoll(payload)
      navigate(`/polls/${created.id}`)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t('create_poll.failed'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <section>
      <h2>{t('create_poll.heading')}</h2>
      <form className="card" onSubmit={onSubmit}>
        {error && <div className="error">{error}</div>}
        <div className="field">
          <label htmlFor="title">{t('create_poll.title')}</label>
          <input id="title" value={title} onChange={(e) => setTitle(e.target.value)} required />
        </div>
        <div className="field">
          <label htmlFor="description">{t('create_poll.description')}</label>
          <textarea id="description" value={description} onChange={(e) => setDescription(e.target.value)} />
        </div>
        <div className="field">
          <label htmlFor="question">{t('create_poll.question')}</label>
          <input id="question" value={question} onChange={(e) => setQuestion(e.target.value)} required />
        </div>
        <div className="checkbox-row">
          <input id="anon" type="checkbox" checked={isAnonymous} onChange={(e) => setIsAnonymous(e.target.checked)} />
          <label htmlFor="anon">{t('create_poll.anonymous')}</label>
        </div>
        <div className="checkbox-row">
          <input
            id="custom"
            type="checkbox"
            checked={allowCustom}
            onChange={(e) => {
              const v = e.target.checked
              setAllowCustom(v)
              if (v) setIsMultiple(false)
            }}
          />
          <label htmlFor="custom">{t('create_poll.free_text')}</label>
        </div>
        <div className="checkbox-row">
          <input
            id="multi"
            type="checkbox"
            checked={isMultiple}
            disabled={allowCustom}
            onChange={(e) => setIsMultiple(e.target.checked)}
          />
          <label htmlFor="multi">{t('create_poll.multiple_choice')}</label>
        </div>
        {isMultiple && (
          <div className="field">
            <label htmlFor="max">{t('create_poll.max_choices')}</label>
            <input
              id="max"
              type="number"
              min={2}
              value={maxChoices}
              onChange={(e) => setMaxChoices(Number(e.target.value))}
            />
          </div>
        )}
        {!allowCustom && (
          <div className="field">
            <label htmlFor="options">{t('create_poll.options')}</label>
            <textarea
              id="options"
              rows={5}
              value={optionsText}
              onChange={(e) => setOptionsText(e.target.value)}
              placeholder={t('create_poll.options_placeholder')}
              required
            />
          </div>
        )}
        <div className="row">
          <div className="field">
            <label htmlFor="start">{t('create_poll.starts')}</label>
            <input id="start" type="datetime-local" value={startAt} onChange={(e) => setStartAt(e.target.value)} required />
          </div>
          <div className="field">
            <label htmlFor="end">{t('create_poll.ends')}</label>
            <input id="end" type="datetime-local" value={endAt} onChange={(e) => setEndAt(e.target.value)} required />
          </div>
        </div>
        <button type="submit" className="btn primary" disabled={busy}>
          {busy ? t('create_poll.submitting') : t('create_poll.submit')}
        </button>
      </form>
    </section>
  )
}
