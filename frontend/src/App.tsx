import { Navigate, NavLink, Outlet, Route, Routes, useNavigate } from 'react-router-dom'
import { AuthProvider, useAuth } from './auth/AuthContext'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import PollListPage from './pages/PollListPage'
import CreatePollPage from './pages/CreatePollPage'
import PollDetailPage from './pages/PollDetailPage'
import PollResultsPage from './pages/PollResultsPage'
import ModerationQueuePage from './pages/ModerationQueuePage'
import ReportDetailPage from './pages/ReportDetailPage'

function userInitial(label: string | null | undefined, fallback: string): string {
  const v = (label ?? '').trim() || fallback
  return v.charAt(0).toUpperCase()
}

function Shell() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="brand">
          <span className="dot" />
          Pollify
        </div>
        <nav>
          <NavLink to="/polls" className={({ isActive }) => (isActive ? 'active' : '')}>
            Polls
          </NavLink>
          <NavLink to="/polls/new" className={({ isActive }) => (isActive ? 'active' : '')}>
            New poll
          </NavLink>
          {user?.role === 'ADMIN' && (
            <NavLink to="/moderation" className={({ isActive }) => (isActive ? 'active' : '')}>
              Moderation
            </NavLink>
          )}
        </nav>
        <div className="user-chip">
          {user ? (
            <>
              <span className="name">
                <span className="avatar">{userInitial(user.display_name || user.email, user.id)}</span>
                {user.display_name || user.email || user.id}
                <span className="tag role">{user.role}</span>
              </span>
              <button
                className="btn ghost"
                onClick={() => {
                  logout()
                  navigate('/login')
                }}
              >
                Sign out
              </button>
            </>
          ) : null}
        </div>
      </header>
      <main>
        <Outlet />
      </main>
    </div>
  )
}

function RequireAuth() {
  const { user, loading } = useAuth()
  if (loading)
    return (
      <p className="muted" style={{ padding: '2rem 1.5rem' }}>
        <span className="spinner" /> Loading…
      </p>
    )
  if (!user) return <Navigate to="/login" replace />
  return <Shell />
}

function RequireAdmin() {
  const { user, loading } = useAuth()
  if (loading)
    return (
      <p className="muted">
        <span className="spinner" /> Loading…
      </p>
    )
  if (!user) return <Navigate to="/login" replace />
  if (user.role !== 'ADMIN') return <Navigate to="/polls" replace />
  return <Outlet />
}

export default function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route element={<RequireAuth />}>
          <Route index element={<Navigate to="/polls" replace />} />
          <Route path="/polls" element={<PollListPage />} />
          <Route path="/polls/new" element={<CreatePollPage />} />
          <Route path="/polls/:pollId" element={<PollDetailPage />} />
          <Route path="/polls/:pollId/results" element={<PollResultsPage />} />
          <Route element={<RequireAdmin />}>
            <Route path="/moderation" element={<ModerationQueuePage />} />
            <Route path="/reports/:reportId" element={<ReportDetailPage />} />
          </Route>
        </Route>
        <Route path="*" element={<Navigate to="/polls" replace />} />
      </Routes>
    </AuthProvider>
  )
}
