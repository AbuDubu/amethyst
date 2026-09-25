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

Requires Go (version in `server/go.mod`), Node (version in `web/.nvmrc`), and make.

```bash
make dev        # build the UI, then run the Go server at http://localhost:8080
make dev-web    # in a second terminal: hot-reloading UI at http://localhost:5173
make check      # every CI check: formatting, static analysis, types, lint, tests, build
make build      # production build: web/dist and server/bin/amethyst
```

The server reads `AMETHYST_ADDR` (default `:8080`) and `AMETHYST_WEB_DIR` (default `web/dist`, relative to the working directory).

## License

[AGPL-3.0](LICENSE)
