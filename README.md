# Amethyst

Amethyst connects communities around shared interests across independently operated servers. It favors chronological community discussions over feeds, trending lists, and engagement metrics.

Each server hosts accounts and communities. Accounts can join communities hosted on other servers through a custom federation protocol. Server operators approve peer connections, and each community separately chooses which peers it shares with.

**Status:** early development. The design baseline is under [`docs/`](docs/):

- [Design brief](docs/design-brief.md): product decisions, architecture, and the draft schema
- [Development workflow](docs/development-workflow.md): milestones, conventions, and verification
- [Glossary](docs/CONTEXT.md): domain language
- [Decision records](docs/adr/)

**Stack:** Go, PostgreSQL, React + TypeScript (Vite), Docker Compose, and Caddy.

This is a solo portfolio and learning project.

## Running locally

Requires Go (version in `server/go.mod`), Node (version in `web/.nvmrc`), Docker, and make.

```bash
make db-up      # start PostgreSQL and Mailpit (captured mail: http://localhost:8025)
make dev        # build the UI, apply migrations, run the Go server at http://localhost:8080
make dev-web    # in a second terminal: hot-reloading UI at http://localhost:5173
make check      # every CI check: formatting, static analysis, types, lint, tests, build
make generate   # regenerate code after editing api/openapi.yaml or a queries.sql file
make jobs       # background job queue: pending/failed counts, most overdue job, recent failures
make build      # production build: web/dist and server/bin/amethyst
```

`make db-reset` deletes all local database data. The server binary has two main commands: `amethyst migrate` applies pending schema migrations, and `amethyst serve` runs HTTP. Migrations never run implicitly; `serve` refuses to start until they are applied.

Configuration (environment variables):

- `AMETHYST_DATABASE_URL` (required)
- `AMETHYST_CANONICAL_ORIGIN` (required): the server's permanent identity, such as `https://example.org`. Plain `http` is allowed only for `localhost`. It is recorded on first start, and the server refuses to start if it later changes.
- `AMETHYST_ADDR` (default `:8080`)
- `AMETHYST_WEB_DIR` (default `web/dist`)

The Makefile supplies local values for the required variables.

### Two servers

```bash
make fed-up     # build the image; run http://a.localhost:8081 and http://b.localhost:8082
make smoke      # check each has its own identity and database, and can reach the other
make fed-down
```

Each server has its own database (`amethyst_a`, `amethyst_b`) on the shared PostgreSQL. `*.localhost` names resolve to your machine in browsers and curl, with no hosts-file changes needed. Tests use `AMETHYST_TEST_DATABASE_URL` to create a throwaway database per test, and fail if it is unset.

## License

[AGPL-3.0](LICENSE)
