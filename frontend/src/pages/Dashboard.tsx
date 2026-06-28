import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'

export function Dashboard() {
  const { data: projects, isLoading, isError } = useQuery({
    queryKey: ['projects'],
    queryFn: api.listProjects,
  })

  return (
    <div>
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-slate-900">Projects</h1>
        <Link
          to="/projects/new"
          className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition hover:bg-indigo-700"
        >
          New project
        </Link>
      </div>

      {isLoading && <p className="mt-6 text-slate-500">Loading…</p>}
      {isError && <p className="mt-6 text-red-600">Failed to load projects.</p>}

      {projects && projects.length === 0 && (
        <div className="mt-6 rounded-lg border border-dashed border-slate-300 bg-white p-10 text-center">
          <p className="text-slate-600">No projects yet.</p>
          <Link to="/projects/new" className="mt-2 inline-block text-sm font-medium text-slate-900 underline">
            Create your first project
          </Link>
        </div>
      )}

      <ul className="mt-6 grid gap-3">
        {projects?.map((p) => (
          <li key={p.id}>
            <Link
              to={`/projects/${p.id}`}
              className="block rounded-xl border border-slate-200 bg-white p-4 transition hover:border-indigo-300 hover:shadow-sm"
            >
              <div className="flex items-center justify-between">
                <span className="font-medium text-slate-900">{p.name}</span>
                <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600">
                  {p.runtime} · {p.environment}
                </span>
              </div>
              <p className="mt-1 truncate text-sm text-slate-500">
                {p.repo_url} ({p.branch})
              </p>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  )
}
