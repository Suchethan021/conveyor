-- Conveyor initial schema.
-- All domain tables carry audit fields; ownership is enforced via owner_id.

CREATE TABLE users (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    github_id  bigint UNIQUE NOT NULL,
    username   text NOT NULL,
    email      text,
    avatar_url text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE projects (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         text NOT NULL,
    git_provider text NOT NULL CHECK (git_provider IN ('github', 'gitlab')),
    repo_url     text NOT NULL,
    branch       text NOT NULL,
    runtime      text NOT NULL CHECK (runtime IN ('go', 'node', 'python', 'static')),
    environment  text NOT NULL CHECK (environment IN ('dev', 'staging', 'prod')),
    created_by   uuid REFERENCES users(id),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_projects_owner ON projects (owner_id);

CREATE TABLE build_jobs (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    status           text NOT NULL DEFAULT 'queued'
                     CHECK (status IN ('queued', 'building', 'scanning', 'deploying', 'success', 'failed', 'cancelled')),
    retry_count      int NOT NULL DEFAULT 0,
    max_retries      int NOT NULL DEFAULT 0,
    failure_reason   text,
    cancel_requested boolean NOT NULL DEFAULT false,
    locked_by        text,
    started_at       timestamptz,
    finished_at      timestamptz,
    created_by       uuid REFERENCES users(id),
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

-- Supports the worker's FOR UPDATE SKIP LOCKED claim ordering.
CREATE INDEX idx_build_jobs_claim ON build_jobs (status, created_at);
CREATE INDEX idx_build_jobs_project ON build_jobs (project_id);

CREATE TABLE build_logs (
    id         bigserial PRIMARY KEY,
    job_id     uuid NOT NULL REFERENCES build_jobs(id) ON DELETE CASCADE,
    stage      text,
    level      text NOT NULL DEFAULT 'info' CHECK (level IN ('info', 'warn', 'error')),
    message    text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_build_logs_job ON build_logs (job_id, id);
