# Conveyor

> A small, self-hostable developer platform. Connect a git repo, trigger a build,
> watch it move through the pipeline, and read the logs — all in one place.

Conveyor is a mini PaaS: you log in, register a project (repo + branch + runtime),
and hit **Build / Deploy**. A Go worker picks the job off a queue and moves it through
`queued → building → scanning → deploying → success / failed`, streaming logs as it goes.

## Stack

| Layer       | Technology                                          |
| ----------- | --------------------------------------------------- |
| Frontend    | Vite + React + TypeScript, Tailwind, TanStack Query |
| Backend     | Go — chi (router), pgx (driver), sqlc (queries)     |
| Database    | PostgreSQL (Docker locally · Neon for the demo)     |
| Migrations  | golang-migrate                                       |
| Auth        | GitHub OAuth (golang.org/x/oauth2) + cookie sessions |
| Deployment  | Docker / Docker Compose · Vercel (FE) · Render (BE) |
| Optional    | Redis, Kubernetes, GitHub Actions                   |

> Full component diagram, schema, and worker design live in [ARCHITECTURE.md](./ARCHITECTURE.md).

## v1 Scope

### Authentication
- OAuth login. **GitHub** is the primary provider.
- After login, the user lands on a dashboard.
- _Roadmap:_ Google and GitLab login.

### Projects
A project is created with:
- Project name
- Git provider: GitHub / GitLab
- Repository URL (validated)
- Branch name
- Runtime type: Go / Node / Python / Static
- Environment: dev / staging / prod

Stored in PostgreSQL, scoped to the owning user.

### Build Worker
A Go worker that processes jobs **asynchronously** and updates status in PostgreSQL.

Job lifecycle: `queued → building → scanning → deploying → success / failed`

Build/deploy stages are simulated with timed steps in v1, but the worker is designed
production-style (real queue semantics, status transitions, structured logs).

_Roadmap / stretch:_
- `SELECT FOR UPDATE SKIP LOCKED` job claiming
- Retry count + failure reason
- Job cancellation
- Worker concurrency limit

### Logs
- Logs stored in PostgreSQL, shown on the job details page.
- _Roadmap:_ live logs via SSE/WebSocket, per-line timestamps, secret/token masking.

### UI Pages
- Login
- Dashboard
- Create project
- Project details
- Build job details (status + logs)

Clean and usable; not heavily designed.

### API
- Current user / session
- Create project
- List projects
- Get project details
- Trigger build job
- Get build job status
- Get build logs

Clear request/response structures and proper error handling throughout.

## Security Principles
- No hardcoded secrets — everything via environment variables.
- Repository URLs are validated.
- Authenticated routes are protected.
- A user can never access another user's projects (ownership enforced at the query layer).
- Sensitive values are masked in logs.
- Audit fields on every record: `created_by`, `created_at`, `updated_at`.

## Deliverables
- This repository
- README with setup instructions
- `docker-compose.yml` for one-command local bring-up
- Architecture notes
- Optional: deployed demo URL

## Layout (planned)

```
conveyor/
├── backend/          # Go API + build worker
├── frontend/         # Next.js app
├── docker-compose.yml
├── SPEC.md           # this file
└── README.md         # setup & run instructions
```
