-- name: InsertAccount :one
INSERT INTO accounts (id, home_server_id, canonical_url, username)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: InsertLocalAccount :one
INSERT INTO local_accounts (account_id, email, password_hash, server_role)
VALUES ($1, $2, $3, $4)
RETURNING *;
