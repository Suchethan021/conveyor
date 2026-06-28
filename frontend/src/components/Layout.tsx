import { Link, Outlet, useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'
import { useMe } from '../lib/auth'
import { Brand } from './Brand'

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
      <header className="sticky top-0 z-10 border-b border-slate-200 bg-white/80 backdrop-blur">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-3">
          <Link to="/" className="rounded-md outline-none focus-visible:ring-2 focus-visible:ring-indigo-500">
            <Brand />
          </Link>
          <div className="flex items-center gap-3 text-sm">
            {user && (
              <span className="flex items-center gap-2">
                {user.avatar_url && (
                  <img src={user.avatar_url} alt="" className="h-6 w-6 rounded-full ring-1 ring-slate-200" />
                )}
                <span className="font-medium text-slate-700">{user.username}</span>
              </span>
            )}
            <button
              onClick={() => logout.mutate()}
              className="rounded-md border border-slate-300 px-3 py-1.5 font-medium text-slate-600 transition hover:bg-slate-50 hover:text-slate-900"
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
