-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByGithubID :one
SELECT * FROM users WHERE github_id = $1;

-- name: UpsertUserByGithubID :one
INSERT INTO users (github_id, username, email, avatar_url)
VALUES ($1, $2, $3, $4)
ON CONFLICT (github_id) DO UPDATE
SET username   = EXCLUDED.username,
    email      = EXCLUDED.email,
    avatar_url = EXCLUDED.avatar_url,
    updated_at = now()
RETURNING *;
