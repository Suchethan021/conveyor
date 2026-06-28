# Conveyor

A small, self-hostable **developer platform** (mini PaaS). Log in, register a project
(repo + branch + runtime), trigger a build, and watch it move through a pipeline with live
logs — `queued → building → scanning → deploying → success / failed`.

Builds are processed by a production-style background worker: jobs are claimed from Postgres
with `FOR UPDATE SKIP LOCKED`, run through timed stages, and support retries, cancellation,
and a worker concurrency limit. Secrets are masked out of logs before they're stored.

> Architecture deep-dive: [ARCHITECTURE.md](./ARCHITECTURE.md) · Product spec: [SPEC.md](./SPEC.md)

## Stack

| Layer    | Technology                                          |
| -------- | --------------------------------------------------- |
| Frontend | Vite + React + TypeScript, Tailwind, TanStack Query |
| Backend  | Go — chi (router), pgx (driver), sqlc (queries)     |
| Database | PostgreSQL 16                                       |
| Auth     | GitHub OAuth + signed-cookie sessions               |
| Runtime  | Docker / Docker Compose                             |

## Quick start (Docker)

Requires only **Docker** (with Compose).

```bash
git clone https://github.com/Suchethan021/conveyor
cd conveyor
cp .env.example .env     # the defaults work out of the box for local use
docker compose up --build
```

This starts four things in order: Postgres → migrations → the Go backend (API + worker) →
the React frontend.

Then open **http://localhost:3000** and click **“Dev login (local)”** — no GitHub setup
needed to explore. Create a project, hit **Build / Deploy**, and watch the logs stream.

| Service  | URL                              |
| -------- | -------------------------------- |
| Frontend | http://localhost:3000            |
| Backend  | http://localhost:8080            |
| Health   | http://localhost:8080/healthz    |

Tear down (and wipe the database volume):

```bash
docker compose down -v
```

### Things to try
- **Happy path:** create a project on branch `main` and build it → ends `success`.
- **Failure + retry:** create a project whose branch name contains `fail` → the scan stage
  fails, the job retries once, then ends `failed` with a failure reason.
- **Cancellation:** trigger a build and hit **Cancel** while it's running.
- **Secret masking:** the logs include a fake `ghp_…` token that is stored as `***REDACTED***`.

## Configuration

All configuration is via environment variables (nothing is hardcoded). Copy `.env.example`
to `.env` and adjust. The committed defaults are **local-only and throwaway** — real secrets
never live in the repo.

| Variable | Purpose | Local default |
| --- | --- | --- |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | Local Postgres container | `conveyor` |
| `SESSION_SECRET` | HMAC key for signing session cookies | insecure dev value |
| `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` | GitHub OAuth app (optional) | blank |
| `GITHUB_CALLBACK_URL` | OAuth callback | `http://localhost:8080/api/auth/github/callback` |
| `FRONTEND_URL` | Post-login redirect target | `http://localhost:3000` |
| `ALLOW_DEV_LOGIN` | Enables the local dev-login shortcut | `true` |
| `WORKER_CONCURRENCY` | Number of worker goroutines | `2` |

### Enabling real GitHub login (optional)
1. Register an OAuth app at <https://github.com/settings/developers>:
   - **Homepage URL:** `http://localhost:3000`
   - **Authorization callback URL:** `http://localhost:8080/api/auth/github/callback`
2. Put the Client ID and Secret in your `.env` (`GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`).
3. `docker compose up --build` and use **Continue with GitHub**.

Without these, login routes return `503` and you use **Dev login** instead.

## Local development (hot reload)

Run the backend in Docker and the frontend with Vite's dev server:

```bash
docker compose up db backend     # Postgres + Go API/worker
cd frontend && npm install && npm run dev   # http://localhost:5173
```

Vite proxies `/api` to the backend, so cookies work the same as in production.
(For real GitHub login in this mode, set `FRONTEND_URL=http://localhost:5173`.)

Regenerate type-safe DB code after changing SQL (no local toolchain needed):

```bash
docker run --rm -v "$PWD/backend:/src" -w /src sqlc/sqlc generate
```

## API

All routes are JSON. Authenticated routes require the session cookie and are scoped to the
logged-in user. Errors use a consistent envelope: `{"error": {"code", "message"}}`.

| Method | Path | Description |
| --- | --- | --- |
| GET  | `/api/me` | Current user |
| GET  | `/api/auth/github/login` | Begin GitHub OAuth |
| GET  | `/api/auth/github/callback` | OAuth callback |
| POST | `/api/auth/logout` | Clear session |
| POST | `/api/auth/dev-login` | Local-only login shortcut |
| POST | `/api/projects` | Create a project |
| GET  | `/api/projects` | List your projects |
| GET  | `/api/projects/{id}` | Project details |
| POST | `/api/projects/{id}/builds` | Trigger a build |
| GET  | `/api/projects/{id}/builds` | List a project's builds |
| GET  | `/api/builds/{id}` | Build job status |
| GET  | `/api/builds/{id}/logs` | Build logs |
| GET  | `/api/builds/{id}/logs/stream` | Live logs via Server-Sent Events |
| POST | `/api/builds/{id}/cancel` | Request cancellation |

## Project structure

```
conveyor/
├── backend/              # Go API + build worker
│   ├── cmd/server/       # entrypoint
│   ├── internal/
│   │   ├── api/          # chi router + handlers
│   │   ├── auth/         # GitHub OAuth + sessions
│   │   ├── worker/       # job claim loop + stages
│   │   ├── db/           # migrations, sqlc queries + generated code
│   │   ├── logsec/       # secret masking
│   │   ├── httpx/        # JSON/error helpers
│   │   └── config/       # env config
│   └── Dockerfile
├── frontend/             # Vite + React + TS SPA
│   ├── src/{pages,components,lib}
│   ├── nginx.conf        # serves SPA + proxies /api
│   └── Dockerfile
├── docker-compose.yml
├── ARCHITECTURE.md
└── README.md
```

## Security notes
- No hardcoded secrets — all config via environment variables.
- Repository URLs are validated (https + provider host + owner/repo).
- Authenticated routes are protected; every query is scoped by `owner_id`, so a user can
  never see another user's projects, builds, or logs (unknown ids return `404`).
- Sensitive values are masked in logs before storage.
- Audit fields (`created_by`, `created_at`, `updated_at`) on all domain tables.
