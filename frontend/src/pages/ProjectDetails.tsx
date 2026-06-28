import { Link, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, isActive } from '../lib/api'
import { StatusBadge } from '../components/StatusBadge'

export function ProjectDetails() {
  const { id = '' } = useParams()
  const qc = useQueryClient()

  const project = useQuery({
    queryKey: ['project', id],
    queryFn: () => api.getProject(id),
  })

  const builds = useQuery({
    queryKey: ['builds', id],
    queryFn: () => api.listBuilds(id),
    // Poll while any build is still in progress.
    refetchInterval: (q) =>
      q.state.data?.some((b) => isActive(b.status)) ? 1500 : false,
  })

  const trigger = useMutation({
    mutationFn: () => api.triggerBuild(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['builds', id] }),
  })

  if (project.isError) {
    return <p className="text-red-600">Project not found.</p>
  }

  return (
    <div>
      <Link to="/" className="text-sm text-slate-500 hover:underline">
        ← Back to projects
      </Link>

      <div className="mt-2 flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-slate-900">
            {project.data?.name ?? '…'}
          </h1>
          {project.data && (
            <p className="mt-1 text-sm text-slate-500">
              {project.data.repo_url} · {project.data.branch} · {project.data.runtime} ·{' '}
              {project.data.environment}
            </p>
          )}
        </div>
        <button
          onClick={() => trigger.mutate()}
          disabled={trigger.isPending}
          className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition hover:bg-indigo-700 disabled:opacity-50"
        >
          {trigger.isPending ? 'Starting…' : 'Build / Deploy'}
        </button>
      </div>

      <h2 className="mt-8 text-lg font-semibold text-slate-900">Builds</h2>

      {builds.isLoading && <p className="mt-3 text-slate-500">Loading…</p>}
      {builds.data && builds.data.length === 0 && (
        <p className="mt-3 text-slate-500">No builds yet — trigger one above.</p>
      )}

      <ul className="mt-3 grid gap-2">
        {builds.data?.map((b) => (
          <li key={b.id}>
            <Link
              to={`/builds/${b.id}`}
              className="flex items-center justify-between rounded-xl border border-slate-200 bg-white px-4 py-3 transition hover:border-indigo-300 hover:shadow-sm"
            >
              <div className="flex items-center gap-3">
                <StatusBadge status={b.status} />
                <span className="font-mono text-xs text-slate-500">{b.id.slice(0, 8)}</span>
                {b.retry_count > 0 && (
                  <span className="text-xs text-amber-600">retry {b.retry_count}</span>
                )}
              </div>
              <span className="text-xs text-slate-400">
                {new Date(b.created_at).toLocaleString()}
              </span>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  )
}
