import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import en from './locales/en'
import ru from './locales/ru'

export type Locale = 'en' | 'ru'

const DICTS: Record<Locale, typeof en> = { en, ru }
const STORAGE_KEY = 'pollify.locale'

function lookup(dict: unknown, path: string): unknown {
  return path.split('.').reduce<unknown>((acc, key) => {
    if (acc && typeof acc === 'object' && key in (acc as Record<string, unknown>)) {
      return (acc as Record<string, unknown>)[key]
    }
    return undefined
  }, dict)
}

function interpolate(s: string, vars?: Record<string, string | number>): string {
  if (!vars) return s
  return Object.keys(vars).reduce(
    (acc, k) => acc.replace(new RegExp(`\\{${k}\\}`, 'g'), String(vars[k])),
    s,
  )
}

interface LocaleState {
  locale: Locale
  setLocale: (l: Locale) => void
  t: (key: string, vars?: Record<string, string | number>) => string
  tn: (key: string, count: number, vars?: Record<string, string | number>) => string
}

const LocaleContext = createContext<LocaleState | undefined>(undefined)

function readStoredLocale(): Locale {
  const v = localStorage.getItem(STORAGE_KEY)
  return v === 'ru' || v === 'en' ? v : 'en'
}

export function LocaleProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(() => {
    try {
      return readStoredLocale()
    } catch {
      return 'en'
    }
  })

  useEffect(() => {
    document.documentElement.lang = locale
  }, [locale])

  const setLocale = useCallback((l: Locale) => {
    try {
      localStorage.setItem(STORAGE_KEY, l)
    } catch {
      // ignore quota / privacy errors
    }
    setLocaleState(l)
  }, [])

  const t = useCallback(
    (key: string, vars?: Record<string, string | number>) => {
      const raw = lookup(DICTS[locale], key)
      if (typeof raw === 'string') return interpolate(raw, vars)
      const fallback = lookup(DICTS.en, key)
      if (typeof fallback === 'string') return interpolate(fallback, vars)
      return key
    },
    [locale],
  )

  const tn = useCallback(
    (key: string, count: number, vars?: Record<string, string | number>) => {
      const rule = new Intl.PluralRules(locale).select(count)
      const candidates = [`${key}.${rule}`, `${key}.other`]
      for (const candidate of candidates) {
        const raw = lookup(DICTS[locale], candidate)
        if (typeof raw === 'string') return interpolate(raw, { count, ...vars })
      }
      const fallback = lookup(DICTS.en, `${key}.other`)
      if (typeof fallback === 'string') return interpolate(fallback, { count, ...vars })
      return `${count} ${key}`
    },
    [locale],
  )

  const value = useMemo<LocaleState>(
    () => ({ locale, setLocale, t, tn }),
    [locale, setLocale, t, tn],
  )

  return <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>
}

export function useT(): LocaleState {
  const ctx = useContext(LocaleContext)
  if (!ctx) throw new Error('useT must be used within LocaleProvider')
  return ctx
}
