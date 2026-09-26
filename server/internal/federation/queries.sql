-- name: CreateServer :one
INSERT INTO servers (canonical_origin, is_local)
VALUES ($1, $2)
RETURNING *;

-- name: GetLocalServer :one
SELECT * FROM servers
WHERE is_local;

-- name: InsertLocalServerIfAbsent :exec
-- Races between concurrent starts are settled by the servers_single_local index.
INSERT INTO servers (canonical_origin, is_local)
VALUES ($1, true)
ON CONFLICT (is_local) WHERE is_local DO NOTHING;
