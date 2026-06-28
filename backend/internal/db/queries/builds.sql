-- name: CreateBuildJob :one
INSERT INTO build_jobs (project_id, created_by, max_retries)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetBuildJobForOwner :one
SELECT bj.* FROM build_jobs bj
JOIN projects p ON p.id = bj.project_id
WHERE bj.id = $1 AND p.owner_id = $2;

-- name: ListBuildJobsForProject :many
SELECT bj.* FROM build_jobs bj
JOIN projects p ON p.id = bj.project_id
WHERE bj.project_id = $1 AND p.owner_id = $2
ORDER BY bj.created_at DESC;

-- name: GetBuildLogsForOwner :many
SELECT bl.* FROM build_logs bl
JOIN build_jobs bj ON bj.id = bl.job_id
JOIN projects p ON p.id = bj.project_id
WHERE bl.job_id = $1 AND p.owner_id = $2
ORDER BY bl.id ASC;

-- name: GetBuildLogsAfterForOwner :many
SELECT bl.* FROM build_logs bl
JOIN build_jobs bj ON bj.id = bl.job_id
JOIN projects p ON p.id = bj.project_id
WHERE bl.job_id = $1 AND p.owner_id = $2 AND bl.id > $3
ORDER BY bl.id ASC;

-- name: RequestCancelForOwner :execrows
UPDATE build_jobs
SET cancel_requested = true, updated_at = now()
FROM projects p
WHERE build_jobs.project_id = p.id
  AND build_jobs.id = $1
  AND p.owner_id = $2
  AND build_jobs.status IN ('queued', 'building', 'scanning', 'deploying');
