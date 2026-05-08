import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { api, clearSession, getStoredUser, getToken, setSession } from '../api/client'
import type { UserSummary } from '../api/types'

interface AuthState {
  user: UserSummary | null
  loading: boolean
  login: (email: string, password: string) => Promise<void>
  register: (email: string, password: string, displayName?: string) => Promise<void>
  logout: () => void
  refresh: () => Promise<void>
}

const AuthContext = createContext<AuthState | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<UserSummary | null>(getStoredUser())
  const [loading, setLoading] = useState<boolean>(Boolean(getToken()))

  useEffect(() => {
    if (!getToken()) {
      setLoading(false)
      return
    }
    api
      .me()
      .then((u) => setUser(u))
      .catch(() => {
        clearSession()
        setUser(null)
      })
      .finally(() => setLoading(false))
  }, [])

  const value = useMemo<AuthState>(
    () => ({
      user,
      loading,
      async login(email, password) {
        const session = await api.login(email, password)
        setSession(session)
        setUser(session.user)
      },
      async register(email, password, displayName) {
        const session = await api.register(email, password, displayName)
        setSession(session)
        setUser(session.user)
      },
      logout() {
        clearSession()
        setUser(null)
      },
      async refresh() {
        if (!getToken()) return
        const u = await api.me()
        setUser(u)
      },
    }),
    [user, loading],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
