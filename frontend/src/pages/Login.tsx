import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api, githubLoginUrl } from '../lib/api'
import { useMe } from '../lib/auth'
import { Brand, GitHubIcon } from '../components/Brand'

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
    <div className="grid min-h-screen place-items-center bg-gradient-to-b from-white to-slate-100 px-4">
      <div className="w-full max-w-sm">
        <Brand className="justify-center" />
        <div className="mt-6 rounded-2xl border border-slate-200 bg-white p-8 shadow-sm">
          <h1 className="text-xl font-semibold text-slate-900">Sign in</h1>
          <p className="mt-1 text-sm text-slate-500">
            Connect a repo, trigger builds, and watch them deploy.
          </p>

          <a
            href={githubLoginUrl}
            className="mt-6 flex w-full items-center justify-center gap-2 rounded-lg bg-slate-900 px-4 py-2.5 font-medium text-white transition hover:bg-slate-800"
          >
            <GitHubIcon />
            Continue with GitHub
          </a>

          {showDevLogin && (
            <>
              <div className="my-4 flex items-center gap-3 text-xs text-slate-400">
                <span className="h-px flex-1 bg-slate-200" />
                or
                <span className="h-px flex-1 bg-slate-200" />
              </div>
              <button
                onClick={() => devLogin.mutate()}
                disabled={devLogin.isPending}
                className="w-full rounded-lg border border-slate-300 px-4 py-2.5 font-medium text-slate-700 transition hover:bg-slate-50 disabled:opacity-50"
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
    </div>
  )
}
