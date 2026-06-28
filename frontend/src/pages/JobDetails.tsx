import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, API_BASE, isActive, type BuildLog } from '../lib/api'
import { StatusBadge } from '../components/StatusBadge'

const levelColor: Record<string, string> = {
  info: 'text-slate-300',
  warn: 'text-amber-300',
  error: 'text-red-400',
}

export function JobDetails() {
  const { id = '' } = useParams()
  const qc = useQueryClient()

  const job = useQuery({
    queryKey: ['build', id],
    queryFn: () => api.getBuild(id),
    refetchInterval: (q) => (q.state.data && isActive(q.state.data.status) ? 1500 : false),
  })

  const active = job.data ? isActive(job.data.status) : false

  // Live logs over Server-Sent Events: the stream sends all existing lines,
  // then tails new ones, and emits a "done" event when the job finishes.
  const [logs, setLogs] = useState<BuildLog[]>([])
  useEffect(() => {
    setLogs([])
    // withCredentials sends the session cookie when the API is on another origin.
    const es = new EventSource(`${API_BASE}/api/builds/${id}/logs/stream`, {
      withCredentials: true,
    })
    es.onmessage = (e) => {
      const line = JSON.parse(e.data) as BuildLog
      setLogs((prev) => [...prev, line])
    }
    es.addEventListener('done', () => {
      es.close()
      qc.invalidateQueries({ queryKey: ['build', id] })
    })
    es.onerror = () => es.close()
    return () => es.close()
  }, [id, qc])

  const cancel = useMutation({
    mutationFn: () => api.cancelBuild(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['build', id] }),
  })

  if (job.isError) {
    return <p className="text-red-600">Build not found.</p>
  }

  return (
    <div>
      <Link
        to={job.data ? `/projects/${job.data.project_id}` : '/'}
        className="text-sm text-slate-500 hover:underline"
      >
        ← Back to project
      </Link>

      <div className="mt-2 flex items-start justify-between">
        <div>
          <h1 className="flex items-center gap-3 text-2xl font-semibold text-slate-900">
            Build {id.slice(0, 8)}
            {job.data && <StatusBadge status={job.data.status} />}
          </h1>
          {job.data && (
            <p className="mt-1 text-sm text-slate-500">
              attempt {job.data.retry_count + 1} of {job.data.max_retries + 1}
              {job.data.started_at && ` · started ${new Date(job.data.started_at).toLocaleTimeString()}`}
              {job.data.finished_at && ` · finished ${new Date(job.data.finished_at).toLocaleTimeString()}`}
            </p>
          )}
          {job.data?.failure_reason && (
            <p className="mt-2 text-sm text-red-600">Failure: {job.data.failure_reason}</p>
          )}
        </div>

        {active && (
          <button
            onClick={() => cancel.mutate()}
            disabled={cancel.isPending || job.data?.cancel_requested}
            className="rounded-md border border-red-300 px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-50 disabled:opacity-50"
          >
            {job.data?.cancel_requested ? 'Cancelling…' : 'Cancel'}
          </button>
        )}
      </div>

      <div className="mt-8 overflow-hidden rounded-xl border border-slate-800 bg-slate-900 shadow-sm">
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-2">
          <span className="text-sm font-medium text-slate-300">Logs</span>
          {active && (
            <span className="flex items-center gap-1.5 text-xs text-emerald-400">
              <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-400" />
              live
            </span>
          )}
        </div>
        <div className="max-h-[28rem] overflow-auto p-4 font-mono text-xs leading-relaxed">
          {logs.length === 0 && <p className="text-slate-500">Waiting for output…</p>}
          {logs.map((line: BuildLog) => (
            <div key={line.id} className="whitespace-pre-wrap">
              <span className="text-slate-500">
                {new Date(line.created_at).toLocaleTimeString()}{' '}
              </span>
              {line.stage && <span className="text-slate-400">[{line.stage}] </span>}
              <span className={levelColor[line.level] ?? 'text-slate-300'}>{line.message}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
