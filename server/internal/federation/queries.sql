-- name: CreateServer :one
INSERT INTO servers (canonical_origin, is_local)
VALUES ($1, $2)
RETURNING *;

-- name: GetLocalServer :one
SELECT * FROM servers
WHERE is_local;
