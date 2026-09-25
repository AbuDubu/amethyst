-- +goose Up

-- Every server this one knows about, including itself. Canonical origins are
-- the stable identity of a server; they are stored normalized (lowercase
-- scheme and host, optional port, no path or trailing slash) so that one
-- server can never appear twice under different spellings.
CREATE TABLE servers (
    id               uuid        PRIMARY KEY DEFAULT uuidv7(),
    canonical_origin text        NOT NULL UNIQUE
        CHECK (canonical_origin ~ '^https?://[a-z0-9.-]+(:[0-9]{1,5})?$'),
    is_local         boolean     NOT NULL DEFAULT false,
    created_at       timestamptz NOT NULL DEFAULT now()
);

-- At most one row describes this server itself. (The database cannot require
-- "at least one"; startup is responsible for creating it.)
CREATE UNIQUE INDEX servers_single_local ON servers (is_local) WHERE is_local;

-- +goose Down
DROP TABLE servers;
