import { Navigate, Outlet } from 'react-router-dom'
import { useMe } from '../lib/auth'

// Gate for authenticated routes: redirects to /login when there's no session.
export function ProtectedRoute() {
  const { isLoading, isError } = useMe()

  if (isLoading) {
    return <div className="grid min-h-screen place-items-center text-slate-500">Loading…</div>
  }
  if (isError) {
    return <Navigate to="/login" replace />
  }
  return <Outlet />
}
