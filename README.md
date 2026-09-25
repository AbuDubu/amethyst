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
make generate   # regenerate Go query code after editing a queries.sql file
make build      # production build: web/dist and server/bin/amethyst
```

`make db-reset` deletes all local database data. The server binary has two commands: `amethyst migrate` applies pending schema migrations, and `amethyst serve` runs HTTP. Migrations never run implicitly; `/api/readyz` reports 503 until they are applied.

Configuration (environment variables): `AMETHYST_DATABASE_URL` (required; the Makefile supplies the local one), `AMETHYST_ADDR` (default `:8080`), `AMETHYST_WEB_DIR` (default `web/dist`). Tests use `AMETHYST_TEST_DATABASE_URL` to create a throwaway database per test, and fail if it is unset.

## License

[AGPL-3.0](LICENSE)
