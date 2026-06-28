-- name: CreateProject :one
INSERT INTO projects (owner_id, name, git_provider, repo_url, branch, runtime, environment, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListProjectsByOwner :many
SELECT * FROM projects
WHERE owner_id = $1
ORDER BY created_at DESC;

-- name: GetProjectForOwner :one
SELECT * FROM projects
WHERE id = $1 AND owner_id = $2;
