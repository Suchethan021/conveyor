import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api, ApiError, type CreateProjectInput } from '../lib/api'

const RUNTIMES = ['go', 'node', 'python', 'static']
const ENVIRONMENTS = ['dev', 'staging', 'prod']
const PROVIDERS = ['github', 'gitlab']

const labelCls = 'block text-sm font-medium text-slate-700'
const inputCls =
  'mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none transition focus:border-indigo-500 focus:ring-2 focus:ring-indigo-100'

export function CreateProject() {
  const navigate = useNavigate()
  const qc = useQueryClient()

  const [form, setForm] = useState<CreateProjectInput>({
    name: '',
    git_provider: 'github',
    repo_url: '',
    branch: 'main',
    runtime: 'go',
    environment: 'dev',
  })

  const set = (k: keyof CreateProjectInput) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
    setForm((f) => ({ ...f, [k]: e.target.value }))

  const create = useMutation({
    mutationFn: () => api.createProject(form),
    onSuccess: (project) => {
      qc.invalidateQueries({ queryKey: ['projects'] })
      navigate(`/projects/${project.id}`)
    },
  })

  return (
    <div className="mx-auto max-w-xl">
      <Link to="/" className="text-sm text-slate-500 hover:underline">
        ← Back to projects
      </Link>
      <h1 className="mt-2 text-2xl font-semibold text-slate-900">New project</h1>

      <form
        onSubmit={(e) => {
          e.preventDefault()
          create.mutate()
        }}
        className="mt-6 grid gap-4 rounded-xl border border-slate-200 bg-white p-6 shadow-sm"
      >
        <div>
          <label className={labelCls}>Project name</label>
          <input className={inputCls} value={form.name} onChange={set('name')} placeholder="My service" required />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className={labelCls}>Git provider</label>
            <select className={inputCls} value={form.git_provider} onChange={set('git_provider')}>
              {PROVIDERS.map((p) => (
                <option key={p} value={p}>{p}</option>
              ))}
            </select>
          </div>
          <div>
            <label className={labelCls}>Branch</label>
            <input className={inputCls} value={form.branch} onChange={set('branch')} required />
          </div>
        </div>

        <div>
          <label className={labelCls}>Repository URL</label>
          <input
            className={inputCls}
            value={form.repo_url}
            onChange={set('repo_url')}
            placeholder="https://github.com/owner/repo"
            required
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className={labelCls}>Runtime</label>
            <select className={inputCls} value={form.runtime} onChange={set('runtime')}>
              {RUNTIMES.map((r) => (
                <option key={r} value={r}>{r}</option>
              ))}
            </select>
          </div>
          <div>
            <label className={labelCls}>Environment</label>
            <select className={inputCls} value={form.environment} onChange={set('environment')}>
              {ENVIRONMENTS.map((env) => (
                <option key={env} value={env}>{env}</option>
              ))}
            </select>
          </div>
        </div>

        {create.isError && (
          <p className="text-sm text-red-600">
            {create.error instanceof ApiError ? create.error.message : 'Could not create project.'}
          </p>
        )}

        <div className="flex justify-end gap-3">
          <Link to="/" className="rounded-md border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50">
            Cancel
          </Link>
          <button
            type="submit"
            disabled={create.isPending}
            className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition hover:bg-indigo-700 disabled:opacity-50"
          >
            {create.isPending ? 'Creating…' : 'Create project'}
          </button>
        </div>
      </form>
    </div>
  )
}
