import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api, githubLoginUrl } from '../lib/api'
import { useMe } from '../lib/auth'

export function Login() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const { data: user } = useMe()

  useEffect(() => {
    if (user) navigate('/', { replace: true })
  }, [user, navigate])

  const devLogin = useMutation({
    mutationFn: api.devLogin,
    onSuccess: (u) => {
      qc.setQueryData(['me'], u)
      navigate('/')
    },
  })

  // Hidden in production builds (set VITE_ENABLE_DEV_LOGIN=false) so the deployed
  // demo only offers real GitHub login.
  const showDevLogin = import.meta.env.VITE_ENABLE_DEV_LOGIN !== 'false'

  return (
    <div className="grid min-h-screen place-items-center px-4">
      <div className="w-full max-w-sm rounded-xl border border-slate-200 bg-white p-8 shadow-sm">
        <h1 className="text-2xl font-semibold text-slate-900">Conveyor</h1>
        <p className="mt-1 text-sm text-slate-500">
          Sign in to manage your projects and builds.
        </p>

        <a
          href={githubLoginUrl}
          className="mt-6 flex w-full items-center justify-center gap-2 rounded-md bg-slate-900 px-4 py-2.5 font-medium text-white hover:bg-slate-800"
        >
          Continue with GitHub
        </a>

        {showDevLogin && (
          <>
            <button
              onClick={() => devLogin.mutate()}
              disabled={devLogin.isPending}
              className="mt-3 w-full rounded-md border border-slate-300 px-4 py-2.5 font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50"
            >
              {devLogin.isPending ? 'Signing in…' : 'Dev login (local)'}
            </button>

            {devLogin.isError && (
              <p className="mt-3 text-sm text-red-600">Dev login failed. Is the backend running?</p>
            )}
          </>
        )}
      </div>
    </div>
  )
}
