import { Link, Outlet, useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'
import { useMe } from '../lib/auth'

export function Layout() {
  const { data: user } = useMe()
  const qc = useQueryClient()
  const navigate = useNavigate()

  const logout = useMutation({
    mutationFn: api.logout,
    onSuccess: () => {
      qc.clear()
      navigate('/login')
    },
  })

  return (
    <div className="min-h-full">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-3">
          <Link to="/" className="text-lg font-semibold text-slate-900">
            Conveyor
          </Link>
          <div className="flex items-center gap-4 text-sm">
            {user && <span className="text-slate-600">{user.username}</span>}
            <button
              onClick={() => logout.mutate()}
              className="rounded-md border border-slate-300 px-3 py-1.5 font-medium text-slate-700 hover:bg-slate-50"
            >
              Log out
            </button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-4 py-8">
        <Outlet />
      </main>
    </div>
  )
}
