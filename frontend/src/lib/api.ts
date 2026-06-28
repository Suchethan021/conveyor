// Typed client for the Conveyor backend. All calls go through /api, which the
// Vite dev server proxies to the Go backend (same-origin, so cookies ride along).

export interface User {
  id: string
  github_id: number
  username: string
  email?: string
  avatar_url?: string
}

export interface Project {
  id: string
  name: string
  git_provider: string
  repo_url: string
  branch: string
  runtime: string
  environment: string
  created_at: string
  updated_at: string
}

export interface BuildJob {
  id: string
  project_id: string
  status: string
  retry_count: number
  max_retries: number
  failure_reason?: string
  cancel_requested: boolean
  started_at?: string
  finished_at?: string
  created_at: string
  updated_at: string
}

export interface BuildLog {
  id: number
  stage?: string
  level: string
  message: string
  created_at: string
}

export interface CreateProjectInput {
  name: string
  git_provider: string
  repo_url: string
  branch: string
  runtime: string
  environment: string
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`/api${path}`, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })

  if (!res.ok) {
    let message = res.statusText
    try {
      const body = await res.json()
      message = body?.error?.message ?? message
    } catch {
      // non-JSON error body; keep statusText
    }
    throw new ApiError(res.status, message)
  }

  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

export const api = {
  // auth
  getMe: () => request<User>('/me'),
  devLogin: () => request<User>('/auth/dev-login', { method: 'POST' }),
  logout: () => request<void>('/auth/logout', { method: 'POST' }),

  // projects
  listProjects: () => request<Project[]>('/projects'),
  getProject: (id: string) => request<Project>(`/projects/${id}`),
  createProject: (input: CreateProjectInput) =>
    request<Project>('/projects', { method: 'POST', body: JSON.stringify(input) }),

  // builds
  listBuilds: (projectId: string) => request<BuildJob[]>(`/projects/${projectId}/builds`),
  triggerBuild: (projectId: string) =>
    request<BuildJob>(`/projects/${projectId}/builds`, { method: 'POST' }),
  getBuild: (id: string) => request<BuildJob>(`/builds/${id}`),
  getBuildLogs: (id: string) => request<BuildLog[]>(`/builds/${id}/logs`),
  cancelBuild: (id: string) => request<void>(`/builds/${id}/cancel`, { method: 'POST' }),
}

// Full-page redirect into the GitHub OAuth flow.
export const githubLoginUrl = '/api/auth/github/login'

// Statuses for which a job is still progressing and worth polling.
export const ACTIVE_STATUSES = ['queued', 'building', 'scanning', 'deploying']
export const isActive = (status: string) => ACTIVE_STATUSES.includes(status)
