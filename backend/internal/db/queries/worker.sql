-- name: ClaimNextBuildJob :one
-- Atomically claim the oldest queued job. FOR UPDATE SKIP LOCKED lets multiple
-- workers/replicas poll concurrently without ever grabbing the same row.
WITH next AS (
    SELECT id FROM build_jobs
    WHERE status = 'queued' AND cancel_requested = false
    ORDER BY created_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE build_jobs bj
SET status = 'building', locked_by = $1, started_at = now(), updated_at = now()
FROM next
WHERE bj.id = next.id
RETURNING bj.*;

-- name: SetBuildJobStatus :exec
UPDATE build_jobs SET status = $2, updated_at = now() WHERE id = $1;

-- name: MarkBuildJobSuccess :exec
UPDATE build_jobs SET status = 'success', finished_at = now(), updated_at = now() WHERE id = $1;

-- name: MarkBuildJobFailed :exec
UPDATE build_jobs SET status = 'failed', failure_reason = $2, finished_at = now(), updated_at = now() WHERE id = $1;

-- name: MarkBuildJobCancelled :exec
UPDATE build_jobs SET status = 'cancelled', finished_at = now(), updated_at = now() WHERE id = $1;

-- name: RequeueBuildJob :exec
UPDATE build_jobs
SET status = 'queued', retry_count = retry_count + 1, locked_by = NULL, started_at = NULL, updated_at = now()
WHERE id = $1;

-- name: IsCancelRequested :one
SELECT cancel_requested FROM build_jobs WHERE id = $1;

-- name: AppendBuildLog :exec
INSERT INTO build_logs (job_id, stage, level, message) VALUES ($1, $2, $3, $4);

-- name: GetProjectByID :one
SELECT * FROM projects WHERE id = $1;
