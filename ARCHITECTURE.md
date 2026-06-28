# Conveyor — Architecture

This document describes how Conveyor is structured: the components, the repository
layout, the data model, the build-worker design, the auth flow, the API surface, and
the deployment topology. It's the reference we build against and the script for the
technical walkthrough.

## 1. Components

```
            ┌─────────────┐        OAuth         ┌──────────────┐
            │  Browser    │◀───────────────────▶ │   GitHub     │
            │ (Vite SPA)  │                       └──────────────┘
            └──────┬──────┘
                   │ JSON over HTTP (cookie session)
                   ▼
            ┌─────────────────────────────────────────────┐
            │              Go service                      │
            │  ┌────────────┐        ┌──────────────────┐  │
            │  │  HTTP API  │        │   Build Worker   │  │
            │  │  (chi)     │  same  │  (goroutine pool)│  │
            │  └─────┬──────┘  proc  └────────┬─────────┘  │
            └────────┼──────────────────────── ┼──────────┘
                     │            pgx           │
                     ▼                          ▼
            ┌─────────────────────────────────────────────┐
            │                PostgreSQL                    │
            │  users · projects · build_jobs · build_logs  │
            └─────────────────────────────────────────────┘
```

**Key decision:** the HTTP API and the build worker live in **one Go binary** but in
**separate packages** (`internal/api`, `internal/worker`). The worker is a pool of
goroutines that claim jobs from Postgres. This keeps the demo deployable on a single
free host, while the clean package boundary means the worker can be split into its own
process/container later with no code rewrite — only a different `main`. Postgres is the
queue, so a split changes nothing about the job protocol.

## 2. Stack

| Concern            | Choice                                             |
| ------------------ | -------------------------------------------------- |
| Frontend           | Vite + React + TypeScript                          |
| Routing (FE)       | react-router-dom                                   |
| Data fetching (FE) | TanStack Query (polling for job status)            |
| Styling            | Tailwind CSS                                        |
| Backend language   | Go                                                  |
| HTTP router        | chi                                                |
| DB driver          | pgx v5                                              |
| Queries            | sqlc (type-safe Go generated from SQL)             |
| Migrations         | golang-migrate                                      |
| Auth               | golang.org/x/oauth2 (GitHub) + signed-cookie sessions |
| Database           | PostgreSQL 16                                       |
| Local orchestration| Docker Compose                                      |

## 3. Repository Layout

```
conveyor/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go            # entrypoint: config → db pool → start API + worker
│   ├── internal/
│   │   ├── api/                   # chi router, handlers, middleware (authn, ownership)
│   │   ├── auth/                  # GitHub OAuth, session create/verify
│   │   ├── config/               # env loading & validation
│   │   ├── db/
│   │   │   ├── migrations/        # golang-migrate SQL files
│   │   │   ├── queries/           # .sql files consumed by sqlc
│   │   │   └── sqlc/              # generated type-safe Go (do not edit by hand)
│   │   ├── domain/               # core types: User, Project, BuildJob, LogLine
│   │   ├── worker/               # claim loop, stage runner, retry/cancel logic
│   │   └── logsec/               # secret/token masking for log lines
│   ├── sqlc.yaml
│   ├── go.mod
│   ├── Dockerfile
│   └── .env.example
├── frontend/
│   ├── src/
│   │   ├── api/                   # typed API client + response types
│   │   ├── components/           # shared UI (StatusBadge, LogViewer, ...)
│   │   ├── pages/                # Login, Dashboard, CreateProject, ProjectDetails, JobDetails
│   │   ├── hooks/                # useProjects, useJob, useAuth ...
│   │   ├── lib/                  # query client, fetch wrapper
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── index.html
│   ├── vite.config.ts
│   ├── package.json
│   ├── Dockerfile
│   └── .env.example
├── docker-compose.yml             # db + backend + frontend for local dev
├── ARCHITECTURE.md
├── SPEC.md
└── README.md
```

## 4. Data Model

All tables carry audit fields. `created_by` references the acting user; ownership is
enforced in every query (a user only ever sees rows they own).

```
users
  id            uuid    pk
  github_id     bigint  unique not null
  username      text    not null
  email         text
  avatar_url    text
  created_at    timestamptz default now()
  updated_at    timestamptz default now()

projects
  id            uuid    pk
  owner_id      uuid    fk → users(id)        -- ownership boundary
  name          text    not null
  git_provider  text    not null  check in ('github','gitlab')
  repo_url      text    not null              -- validated before insert
  branch        text    not null
  runtime       text    not null  check in ('go','node','python','static')
  environment   text    not null  check in ('dev','staging','prod')
  created_by    uuid    fk → users(id)
  created_at    timestamptz default now()
  updated_at    timestamptz default now()

build_jobs
  id              uuid    pk
  project_id      uuid    fk → projects(id)
  status          text    not null default 'queued'
                  check in ('queued','building','scanning','deploying','success','failed','cancelled')
  retry_count     int     not null default 0
  max_retries     int     not null default 0
  failure_reason  text
  cancel_requested boolean not null default false
  locked_by       text                          -- worker id holding the job
  started_at      timestamptz
  finished_at     timestamptz
  created_by      uuid    fk → users(id)
  created_at      timestamptz default now()
  updated_at      timestamptz default now()

build_logs
  id         bigserial pk
  job_id     uuid    fk → build_jobs(id)
  stage      text                              -- building/scanning/deploying/...
  level      text    default 'info'            -- info/warn/error
  message    text    not null                  -- secrets masked at write time
  created_at timestamptz default now()         -- per-line timestamp

-- indexes
build_jobs(status, created_at)   -- worker claim ordering
build_logs(job_id, id)           -- ordered log fetch
projects(owner_id)
```

Enums are modeled as `text + CHECK` rather than Postgres `ENUM` types, so adding a value
is a plain migration (no `ALTER TYPE` dance).

## 5. Build Worker Design

The worker is a pool of `N` goroutines (concurrency limit from config). Each goroutine
polls Postgres and atomically claims one queued job using **`FOR UPDATE SKIP LOCKED`**,
so multiple workers/replicas never grab the same job:

```sql
WITH next AS (
  SELECT id FROM build_jobs
  WHERE status = 'queued' AND cancel_requested = false
  ORDER BY created_at
  FOR UPDATE SKIP LOCKED
  LIMIT 1
)
UPDATE build_jobs j
SET status = 'building', locked_by = $1, started_at = now(), updated_at = now()
FROM next
WHERE j.id = next.id
RETURNING j.*;
```

Once claimed, the job advances through stages (`building → scanning → deploying →
success`). In v1 each stage is a timed simulation that emits log lines. The design is
production-shaped:

- **Status transitions** are persisted after each stage so the UI reflects live progress.
- **Cancellation:** between stages the worker checks `cancel_requested`; if set, it
  transitions the job to `cancelled` and stops.
- **Retry:** on a stage failure, if `retry_count < max_retries`, the job is re-queued
  (`status='queued'`, `retry_count++`); otherwise it goes `failed` with `failure_reason`.
- **Concurrency limit:** the number of worker goroutines caps in-flight jobs.
- **Crash safety (roadmap):** a reaper requeues jobs whose `locked_by` is stale (worker
  died mid-build) past a heartbeat timeout.

**Simulating outcomes (demo aid):** stages are timed sleeps. To demo the failure/retry
path deterministically, a build whose project **branch name contains `fail`** fails at the
scan stage; every other branch succeeds. The worker also emits one log line containing a
fake `ghp_…` token to show that `internal/logsec` masks secrets before they are stored.

## 6. Authentication Flow

GitHub OAuth (Authorization Code):

1. FE hits `GET /api/auth/github/login` → backend redirects to GitHub with `state`.
2. GitHub redirects back to `GET /api/auth/github/callback?code&state`.
3. Backend exchanges `code` for a token, fetches the GitHub user, upserts into `users`.
4. Backend issues a **signed, httpOnly, SameSite cookie** session and redirects to the SPA.
5. Every protected route runs auth middleware that resolves the session → `user_id`, and
   ownership middleware ensures the requested resource belongs to that user.

Google / GitLab are additional providers behind the same interface (roadmap).

## 7. API Surface

| Method | Path                          | Purpose                       |
| ------ | ----------------------------- | ----------------------------- |
| GET    | `/api/auth/github/login`      | Begin GitHub OAuth            |
| GET    | `/api/auth/github/callback`   | OAuth callback → set session  |
| POST   | `/api/auth/logout`            | Clear session                 |
| GET    | `/api/me`                     | Current user / session        |
| POST   | `/api/projects`               | Create project                |
| GET    | `/api/projects`               | List caller's projects        |
| GET    | `/api/projects/{id}`          | Project details (owner-only)  |
| POST   | `/api/projects/{id}/builds`   | Trigger a build job           |
| GET    | `/api/builds/{id}`            | Build job status              |
| GET    | `/api/builds/{id}/logs`       | Build logs (paginated)        |
| POST   | `/api/builds/{id}/cancel`     | Request cancellation          |
| GET    | `/api/builds/{id}/logs/stream`| Live logs via Server-Sent Events |

Responses use a consistent envelope and structured errors (`{ "error": { "code", "message" } }`)
with appropriate HTTP status codes.

## 8. Deployment Topology

**Local (Docker Compose):** three services — `db` (Postgres), `backend` (Go API + worker),
`frontend` (Vite build served by nginx). One command brings the whole stack up.

**Demo (free tier):**

| Piece    | Host                          | Notes                                         |
| -------- | ----------------------------- | --------------------------------------------- |
| Database | Neon (serverless Postgres)    | Free tier, no card                            |
| Backend  | Render / Fly.io               | API + worker in one process                   |
| Frontend | Vercel                        | Static Vite build; `VITE_API_URL` → backend   |

VIBSL will additionally deploy to their own staging (`app-dev.vibsl.com`) after review,
so our demo deployment is a bonus rather than the primary target.

## 9. Security

- **No hardcoded secrets** — all config via env vars; `.env.example` documents the shape.
- **Repo URL validation** — scheme/host allowlist (https + github.com/gitlab.com) before insert.
- **Protected routes** — session middleware on everything except auth + health.
- **Tenant isolation** — every project/job/log query is filtered by `owner_id`; no
  endpoint trusts a client-supplied user id.
- **Log masking** — `internal/logsec` redacts token/secret-shaped substrings before any
  log line is persisted.
- **Audit fields** — `created_by`, `created_at`, `updated_at` on all domain tables.
```
