# Conveyor

A small, self-hostable **developer platform** (mini PaaS). Log in, register a project
(repo + branch + runtime), trigger a build, and watch it move through a pipeline with live
logs — `queued → building → scanning → deploying → success / failed`.

Builds are processed by a production-style background worker: jobs are claimed from Postgres
with `FOR UPDATE SKIP LOCKED`, run through timed stages, and support retries, cancellation,
and a worker concurrency limit. Secrets are masked out of logs before they're stored.

> Architecture deep-dive: [ARCHITECTURE.md](./ARCHITECTURE.md) · Product spec: [SPEC.md](./SPEC.md)

## Live demo

**▶ https://conveyor-neon.vercel.app** — sign in with GitHub and try it.

> Hosted on free tiers: frontend on **Vercel**, backend (API + worker) on **Render**, database on
> **Neon**. The Render instance sleeps when idle, so the **first request may take ~30–50s to wake**.
> The deployed build uses real GitHub login only (the local dev-login shortcut is disabled).
> Prefer to run it yourself? See [Quick start](#quick-start-docker) below.

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
cp backend/.env.example backend/.env      # backend config & secrets
cp frontend/.env.example frontend/.env    # frontend config (defaults are fine)
docker compose up --build
```

The committed example values work out of the box for local use. The database runs with
built-in throwaway credentials, so no extra setup is needed.

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

Configuration is via environment variables (nothing is hardcoded). Each service owns its own
env file — copy the `.env.example` next to each one. Real secrets live in the gitignored
`.env` files, never in the repo; the committed examples are placeholders / local-only.

**`backend/.env`** — the Go API + worker:

| Variable | Purpose | Local default |
| --- | --- | --- |
| `DATABASE_URL` | Postgres connection (host `db` in compose) | `postgres://conveyor:conveyor@db:5432/conveyor` |
| `SESSION_SECRET` | HMAC key for signing session cookies | dev value (replace it) |
| `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` | GitHub OAuth app (optional) | blank |
| `GITHUB_CALLBACK_URL` | OAuth callback (through the frontend origin) | `http://localhost:3000/api/auth/github/callback` |
| `FRONTEND_URL` | Post-login redirect target | `http://localhost:3000` |
| `ALLOW_DEV_LOGIN` | Enables the local dev-login shortcut | `true` |
| `WORKER_CONCURRENCY` | Number of worker goroutines | `2` |

**`frontend/.env`** — the SPA:

| Variable | Purpose | Local default |
| --- | --- | --- |
| `VITE_API_URL` | Backend base URL; blank = same-origin via proxy | blank |

The Postgres container itself uses built-in local credentials (`conveyor`/`conveyor`) defined
in `docker-compose.yml` — fine for local, since that database is ephemeral and not exposed.

### Enabling real GitHub login (optional)
1. Register an OAuth app at <https://github.com/settings/developers>:
   - **Homepage URL:** `http://localhost:3000`
   - **Authorization callback URL:** `http://localhost:3000/api/auth/github/callback`
     (the callback goes through the frontend so the whole flow stays on one origin and the
     session cookie is set correctly; nginx proxies `/api` to the backend.)
2. Put the Client ID and Secret in `backend/.env` (`GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`),
   and set `GITHUB_CALLBACK_URL=http://localhost:3000/api/auth/github/callback`.
3. `docker compose up --build` and use **Continue with GitHub**.

(For the Vite dev server instead, use `http://localhost:5173/...` in both places.)

Without these, login routes return `503` and you use **Dev login** instead.

## Local development (hot reload)

Run the backend in Docker and the frontend with Vite's dev server:

```bash
docker compose up db backend     # Postgres + Go API/worker
cd frontend && npm install && npm run dev   # http://localhost:5173
```

Vite proxies `/api` to the backend, so cookies work the same as in production.
(For real GitHub login in this mode, set `FRONTEND_URL=http://localhost:5173` in `backend/.env`.)

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
