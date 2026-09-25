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

## License

[AGPL-3.0](LICENSE)
