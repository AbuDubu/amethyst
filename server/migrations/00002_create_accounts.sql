-- +goose Up

-- Account identities known to this server: local accounts (homed here) and,
-- from Milestone 4, remote accounts from peer servers. Nothing private lives
-- here; this is what peers may see.
CREATE TABLE accounts (
    id             uuid        PRIMARY KEY,
    home_server_id uuid        NOT NULL REFERENCES servers (id),
    -- ID-based, so identity never depends on the (currently permanent) username.
    canonical_url  text        NOT NULL UNIQUE,
    -- Stored lowercase, so uniqueness below is case-insensitive.
    username       text        NOT NULL CHECK (username ~ '^[a-z][a-z0-9_]{2,29}$'),
    display_name   text        CHECK (char_length(display_name) BETWEEN 1 AND 64),
    status         text        NOT NULL DEFAULT 'active' CHECK (status IN ('active')),
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (home_server_id, username)
);

-- Private data for local accounts only. Never serialized to peers.
-- Invariant enforced by the identity module (it spans tables): the account's
-- home server is the local server.
CREATE TABLE local_accounts (
    account_id        uuid        PRIMARY KEY REFERENCES accounts (id) ON DELETE CASCADE,
    email             text        NOT NULL CHECK (char_length(email) BETWEEN 3 AND 254 AND email LIKE '_%@_%'),
    email_verified_at timestamptz,
    -- argon2id in PHC string format, including its parameters.
    password_hash     text        NOT NULL CHECK (password_hash LIKE '$argon2id$%'),
    server_role       text        NOT NULL DEFAULT 'member' CHECK (server_role IN ('member', 'operator')),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

-- Email addresses are compared case-insensitively but sent as entered.
CREATE UNIQUE INDEX local_accounts_email_key ON local_accounts (lower(email));

-- +goose Down
DROP TABLE local_accounts;
DROP TABLE accounts;
