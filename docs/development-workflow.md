# Development workflow

## Confirmed requirements

Development is a collaborative learning process. The owner must understand the system as it grows, rather than receive a completed implementation without context.

- Track work in tickets.
- Make coherent, reasonably small commits. Larger initial setup commits are acceptable when they represent a reviewable piece of work.
- Include automated tests, CI/CD, and distinct development and test environments.
- Explain design choices, important code paths, and test evidence throughout implementation.
- Finish the design interview and confirm shared understanding before beginning implementation (completed 2026-09-25).
- Keep planning documentation in `docs/`; update existing files and use ADRs only for significant trade-offs.
- Use a public GitHub monorepo, Issues, pull requests, and GitHub Actions, with short-lived branches and required CI checks.
- Review the intended design and completed increment together for the first few tickets. Reduce checkpoints only at the owner's direction.
- Use local development, disposable CI environments, staging with two logical servers for federation testing, and production. Machine allocation is not yet decided.
- Build an artifact once, verify it in staging, and promote that same artifact to production with the owner's approval initially.

## Agreed working loop

1. Define a ticket with the user-visible outcome, acceptance criteria, important failure cases, and learning objective.
2. Explain the intended design and the small portion of the system that will change.
3. Implement one coherent increment with appropriate tests; avoid combining unrelated refactoring with feature work.
4. Run required checks and present a reviewable change, demonstration, and explanation of the important trade-offs.
5. Walk through the result and tests with the owner before moving into the next learning increment, at the agreed cadence.

The browser/backend interface is an OpenAPI contract (`api/openapi.yaml`) kept in version control, from which Go server code and TypeScript types are generated; CI fails if generated code is stale. Browser API paths carry no version (`/api/...`) because the frontend and backend always ship as one artifact. Errors use RFC 9457 Problem Details (`application/problem+json`) with an added stable `code`. Federation has its own contract with versioned paths (`/federation/v1/...`) and compatibility tests, because independently operated servers run different releases. Explain changes to either contract in the implementing ticket.

## Proposed verification

- Focused tests for domain rules and authorization through module interfaces.
- PostgreSQL integration tests for persistence and transactions.
- Frontend tests of visible behavior and user interactions.
- Browser tests for a small set of critical journeys.
- Independent-server tests for federation, including duplicate deliveries, outages, and recovery.
- CI checks for formatting, static analysis, types, tests, and builds; exact tools remain open.

Important agreed product rules that these tests must cover include current host authorization for cached private reads, unavailable private reads during host outages, fixed community access presets after first publication, independently retained bans and peer-access restrictions, membership access to earlier community history, password-reset session invalidation, and expiration of reported evidence within 30 days of capture.

## Agreed runtime and recovery baseline

- Docker Compose manages PostgreSQL and a local mail capture service during development; Go and frontend development servers can run directly for fast feedback. An optional federation profile starts two logical servers with independent databases and identities.
- Deployment uses a built application image behind Caddy, plus PostgreSQL and a persistent media volume. No application builds run on the production VPS.
- CI uses isolated disposable databases and peer instances, runs formatting/static analysis/types/tests/build checks, verifies migrations and generated contracts, and publishes an artifact only after required checks pass.
- Staging runs the same artifact intended for production. Promotion is manual initially; migrations have an explicit compatibility and recovery plan before deployment.
- Backups are daily database-aware PostgreSQL backups plus media, encrypted and copied off the application VPS. Use a rolling seven-calendar-day retention window, with automated expiry and a documented restore drill. Retention must not quietly extend just because no new backups were produced.
- A daily backup implies potential loss of changes since the latest successful backup. The available destination, storage allowance, and tested recovery procedure must be confirmed before deployed acceptance testing. Eventual real-user adoption is optional.
- Health/readiness endpoints and structured logs should expose delivery failures, pending-work age, backup failure, and storage pressure without logging credentials, private content, or recovery links. No dedicated monitoring cluster is proposed initially.

## Repository conventions

- Public repository `AbuDubu/amethyst`, default branch `main`, licensed AGPL-3.0.
- Layout: `server/` (Go module: `cmd/amethyst`, `internal/` modules plus shared `platform`, `migrations/`), `web/` (Vite/React/TypeScript), `api/` (`openapi.yaml`, the source of truth for generated Go and TypeScript types), `deploy/` (Compose, Caddy, backup scripts), `docs/`, `.github/`. The root `README.md` is the only Markdown file outside `docs/` and `.github/`.
- GitHub Milestones mirror the milestone plan. Full issues are written for the current milestone; the next milestone's issues are written as the current one nears completion.
- Labels: `type:feature`, `type:bug`, `type:chore`, `type:docs`, `area:server`, `area:web`, `area:federation`, `area:infra`.
- Branches are named `<issue#>-short-slug`. Commit subjects are imperative and reference the issue.
- `main` is protected: pull requests and passing required checks, no force pushes. Pull requests merge by rebase so each small commit is preserved; squash and merge commits are disabled.
- Required CI jobs for Milestone 0: **server** (gofmt, go vet, staticcheck, `go test` against PostgreSQL), **web** (tsc, oxlint, vitest, Vite build), and **contract** (regenerate OpenAPI and sqlc output; fail on any difference). Browser-journey and two-server federation jobs become required in the milestones that introduce them. Deployment jobs wait for a VPS provider.

## Environment questions still open

- Exact Compose topology and configuration for local PostgreSQL, mail capture, and isolated peer instances.
- Shared staging resource allocation and whether services run continuously or on demand.
- Exact deployment triggers, migration checks, and rollback strategy.

## Milestone plan

These are the GitHub Milestones. Each group is split into reviewable issues; the owner reviews the intended design and completed increment for the first few issues.

| Milestone | Demonstrable outcome | Learning focus and acceptance evidence |
| --- | --- | --- |
| 0. Foundation | One command starts local dependencies; Go serves a React page and exposes readiness; two logical server instances with independent identities and databases run locally and in CI; CI runs. | Repository layout, builds, migrations, generated contracts, test isolation, and one PostgreSQL integration test. |
| 1. Local identity | Invitation registration, verified email, login/logout, recovery, and session revocation work. | Authentication versus authorization; replayed tokens fail; local mail capture prevents accidental real email. |
| 2. Local communities | Create communities, apply admission modes, assign moderators, and enforce access presets and restrictions. | State transitions and relational constraints; tests prove different roles cannot acquire one another's permissions. |
| 3. Local discussion product | Titled Markdown discussions, processed images, replies, edits, deletion, bookmarks, notifications, and internal search work. | UI state versus server state, authorization-aware search/media, accessible interactions, and content lifecycle. |
| 4. Federation foundation | Two independent identities establish mutual peering, explicit community grants, signed requests, and protocol compatibility tests. | Trust bootstrap, signatures, actor binding, replay checks, revisions, and durable delivery intent. |
| 5. Remote participation | Join, publish, reply, edit, moderate, and recover after outages across two servers. | Idempotency, pending versus confirmed actions, authorization of cached private reads, and revoked sharing. |
| 6. Lifecycle and operations | Account closure, ownership transfer/archive, evidence expiry, staged deployment, and backup restoration work. | Cross-server cleanup, temporal policies, migration safety, and recovery limitations. |
| 7. Working-product review | The complete acceptance suite passes, UI works on mobile and keyboard, and the design can be explained and demonstrated. | Reproducible evidence and a documented architecture walkthrough; real users are not a requirement. |

Build federation fixtures and contract tests while establishing the foundation so federation is exercised before the entire UI is polished. The order above describes dependency milestones, not permission to leave core distributed risks untested until the end.

## Proposed foundation ticket

**Outcome:** a minimal walking skeleton demonstrates the complete local build/test path without implementing social features.

Acceptance criteria:

- The chosen repository layout separates frontend, Go application modules, contract definitions, migrations, operational configuration, and `docs/`.
- Local dependencies start reproducibly and development email stays in a capture service.
- A minimal browser page reaches the Go API; readiness reflects actual database availability.
- One migration and an isolated database integration test demonstrate the persistence workflow.
- Two logical server instances, each with its own database and canonical origin, start locally and in CI, keeping local-only identity assumptions out of the schema.
- CI checks Go formatting/static analysis/tests, frontend types/build/tests, and generated-contract consistency as applicable to the skeleton.
- The change is reviewed with the owner, explaining the request path, dependency choices, and how to run/debug tests.

Suggested commits: repository/build configuration; database/migration/test harness; API/frontend contract connection; CI and run instructions. Keep each commit coherent and buildable where practical.

## Working-product acceptance gate

- Two-server participation succeeds, including duplicate delivery, reordered/stale updates, outages, and recovery.
- Bans, membership changes, and revoked sharing cannot be bypassed by cached content, media links, search results, or pending work.
- Host outages deny private reads while handling public cached content appropriately.
- Account recovery and both closure options behave as documented; remote deletion failures remain visible.
- Evidence is unavailable after its expiry even if the cleanup worker has not yet removed the bytes.
- A database/media backup is restored successfully under the agreed restoration policy.
- Required browser journeys work at mobile widths and with keyboard navigation; known limitations are documented.
- The owner can follow the core request, authorization, transaction, and delivery paths and reproduce the checks.

The goal is a complete working product and learning outcome. A real-user pilot is optional and does not define completion.

Work is tracked at https://github.com/AbuDubu/amethyst. No infrastructure has been provisioned.
