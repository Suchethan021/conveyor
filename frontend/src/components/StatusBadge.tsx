const colors: Record<string, string> = {
  queued: 'bg-slate-100 text-slate-600',
  building: 'bg-blue-100 text-blue-700',
  scanning: 'bg-indigo-100 text-indigo-700',
  deploying: 'bg-violet-100 text-violet-700',
  success: 'bg-green-100 text-green-700',
  failed: 'bg-red-100 text-red-700',
  cancelled: 'bg-amber-100 text-amber-700',
}

export function StatusBadge({ status }: { status: string }) {
  const cls = colors[status] ?? 'bg-slate-100 text-slate-600'
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${cls}`}>
      <span className="h-1.5 w-1.5 rounded-full bg-current opacity-70" />
      {status}
    </span>
  )
}
