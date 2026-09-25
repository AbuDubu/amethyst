# Amethyst design brief

## Confirmed direction

- This is a solo portfolio project and an experiment in AI-assisted programming, not a course submission.
- The attached CMPUT 404 project document is inspiration. Its stack, infrastructure, protocol, feature, and grading requirements are not binding.
- The owner intends to operate deployment using free-tier VPS nodes. GitHub Actions is selected for CI/CD. Provider, resource limits, and availability expectations remain unspecified.
- The project should teach a modern frontend and backend stack and demonstrate deliberate design decisions and backend engineering.
- Go is the accepted backend language. The owner values its simplicity and performance characteristics; no workload-specific performance claims have been established.
- The owner delegated the frontend choice, prioritizing maintainability and testability. Selected direction: React with TypeScript, Vite, React Router, TanStack Query for API-backed state, and CSS Modules with shared design tokens. The initial UI is client-rendered, with Go serving the built assets and public link metadata; extensive search-engine optimization is deferred. Transient UI/form state stays local to the relevant feature.
- PostgreSQL is selected for relational data and durable delivery work. Deployment sizing and configuration remain open.
- The intended audience is people dissatisfied with hype, AI-generated filler, and heavily manipulated presentation on traditional social media.
- A server hosts multiple communities. Each account has a home server and can participate in communities hosted on other servers.
- Account handles have a fixed username and home server in the first release, with editable display names and avatars. Account migration is deferred. Login initially uses local username/password credentials and revocable server-side browser sessions carried in secure cookies. A verified email address stays private on the home server and supports expiring, single-use password recovery links. Password resets invalidate existing sessions. Email delivery must be included in deployment and isolated in tests.
- Account closure offers two choices: retain contributions under a deleted-member identity or request their deletion, with deletion the default. Remove the profile, credentials, email, and sessions while keeping minimal identity placeholders required for references. Owners must transfer ownership or archive their communities first. Remote deletion remains visibly unresolved when a host cannot be reached or no longer accepts communication.
- Server registration supports open enrollment, operator approval, and invitation-only modes. The pilot starts with invitations. A server invitation does not confer community membership.
- Approved local accounts can create communities and become their owners. Owners manage settings, moderator appointments, and community sharing approvals. Moderators manage membership requests, reports, content removals, and bans. Server operators may suspend hosted communities and disconnect peers.
- Community discussions are the core interaction, rather than personal status feeds.
- First-release content includes titled discussions with Markdown text and optional images, threaded replies, chronological ordering, bookmarks, and in-app notifications for replies and membership decisions. Reactions, polls, direct messages, and video uploads are deferred.
- Images are processed and stored on persistent server storage, with metadata in PostgreSQL. Initial configurable limits are four images per discussion and 10 MB per upload, plus decoded-image size limits. Oversized images are resized and embedded metadata removed; original full-resolution files are not retained as a product feature. Restricted images require authorization. Object storage is deferred behind a small storage interface.
- Deleting a contribution preserves the thread structure and other people's replies, replacing the deleted contribution with a placeholder. Its authored content and attachments are removed from active storage, with deletion requested from peers and minimal audit metadata retained. Backup retention and deletion of operational copies still need detailed design.
- Reported content has a disclosed exception: a restricted evidence snapshot may remain accessible to authorized moderators/operators for at most 30 days after capture, then expires. Minimal action metadata is retained separately. This does not authorize general retention of deleted content or extension of the snapshot's lifetime merely by keeping a report open.
- Edits show an edited marker. Full revision history is outside the current scope.
- Community access presets are accepted: public communities are discoverable and readable by anyone, with open or approval-based admission; private communities expose their name and description but restrict content to members admitted by approval; invitation-only communities are unlisted and restrict content to invited members. Posting and commenting require membership; private membership lists are restricted.
- A community's access preset becomes fixed when its first discussion is published in version one. Membership decisions and peer sharing approvals remain changeable.
- Members can access community discussions from before they joined. Eligible members on a newly approved peer can also access history. Fetch older content as needed rather than immediately copying the entire archive to a new peer.
- Compliant peers require fresh authorization from the community host before serving private or invitation-only content, including cached content. If the host cannot be reached, these reads are unavailable. Public cached discussions can remain readable. This does not recall previously downloaded data or prevent copying by a remote operator.
- Public webpages remain anonymously readable even when a community has not opted into federation with a particular server. Federation restrictions govern delivery and remote participation, not the ability to read or copy public webpages.
- Operators on both servers must approve a peer connection. Each community separately opts in to sharing with a particular peer; server approval alone does not grant access to community data. Community sharing is directional and does not require reciprocal community sharing.
- Revoking community sharing stops new deliveries and causes the community host to reject subsequent community actions from that peer. Historical contributions and membership records remain, but access through that peer is suspended. A narrow revocation-cleanup exception (below) still allows deletion requests, deletion markers, and revocation notices. Restoring sharing does not automatically reactivate affected memberships; reactivation requires an explicit decision. Previously delivered copies remain outside the host's control.
- The community's host server owns authoritative discussions and membership state. An author's home server owns their identity and submits authenticated actions to the community host. Authors can request edits or deletion; moderators may remove content but may not rewrite it under an author's name. Publication is not confirmed until the community host accepts it.
- Remote operators may retain copies of received content. The product cannot guarantee remote deletion, including deletion from an operator's independent copies.
- The product favors chronological discussions and avoids a global trending feed. The owner supports these principles alongside mixed community access modes.
- Discovery prioritizes the internal network: searchable local communities and communities explicitly shared by approved peers, plus search over discussions the viewer may access. Invitation-only communities are excluded from directories. Public URLs and public link previews are supported; private links expose only generic previews. Extensive external search-engine optimization is deferred.
- Authenticity policy targets deceptive presentation, spam, and mass-produced engagement bait. Communities may adopt stricter rules, supported by reporting and human moderation; automated authenticity detection is not a product promise.
- The design direction is a neighborhood clubhouse with forum readability: warm neutrals, restrained color, comfortable typography, and community identity emphasized over popularity metrics.
- Initial federation uses a custom protocol between instances of this platform. Supporting other protocols is a future possibility, not a first-release commitment or a guarantee of easy integration.
- Easy testing is a core design requirement across frontend and backend.
- Architecture is one modular Go application per server, with identity, communities and membership, discussions, moderation, and federation modules. A background delivery worker initially runs in that application. Domain changes and their outgoing delivery records are saved in one PostgreSQL transaction, then delivered asynchronously with retries.
- The project aims for a complete, demonstrably working product whether or not real users ever join. Its federation milestone covers two independently deployed servers: join a remote community, publish, reply, edit, moderate, and recover delivery after an outage. A real-user pilot is optional and is not required for project completion.
- The working-product gate includes repeatable permission and recovery tests, account recovery/deletion, evidence expiry, duplicate delivery handling, a successful backup restoration, mobile usability, and keyboard navigation.
- Development uses a public GitHub monorepo, Issues, pull requests, GitHub Actions, short-lived branches, and required CI checks. Commits should be coherent and reasonably small; larger initial setup commits are acceptable when reviewable.
- The owner reviews both the intended design and the completed increment for the first few tickets. Checkpoints may be reduced later at the owner's direction. Explanations, test evidence, and demonstrations support learning throughout development.
- Environment structure is accepted: local development, disposable CI environments, staging with two logical servers for federation tests, and production. Test a built artifact in staging before promoting that same artifact to production; production deployment initially requires the owner's approval. Actual machine allocation awaits provider and capacity information.
- Docker Compose manages reproducible environments. Deployment uses Caddy for HTTPS, persistent PostgreSQL/media storage, and daily encrypted backups copied off the application VPS with rolling seven-calendar-day retention. Restore testing is required. Data since the latest successful backup may be lost; available destinations and capacity must be verified before deployment.
- Keep planning documents under `docs/`, updating existing documents rather than creating one for each interview round. The glossary is `docs/CONTEXT.md`; significant decision records stay in `docs/adr/`.
- The design was finalized as the implementation baseline on 2026-09-25. Implementation proceeds ticket by ticket under the working loop in `docs/development-workflow.md`.
- The Clubhouse layout is selected: a persistent navigation rail on desktop that collapses to compact top navigation on narrow screens.

## Design tree

1. Product scope and success criteria
   - Confirmed: portfolio and learning; solo operation; assignment constraints waived.
   - Confirmed: cross-server discussion milestone and working-product completion gate independent of eventual adoption; learning and disciplined development are required.
   - Deferred: launch audience and calendar dates. Work is milestone-driven; no deadline has been supplied.
2. Community and trust model
   - Confirmed: multiple communities per server, home-server accounts, cross-server participation, access presets, community-host authority, and community moderation of authenticity rules.
   - Confirmed: anonymous public web access is separate from federation; mutual server peering and directional community sharing; revocation suspends affected access and requires explicit reactivation.
   - Confirmed: registration modes, fixed handles with editable profiles, local password login with server-side sessions, community owner/moderator/member roles, and server operator authority.
   - Confirmed: verified-email recovery, fresh host authorization for private reads, historical access for members, fixed access preset after first publication, and at most 30 days of restricted reported-content evidence.
   - Confirmed: both account-closure options, defaulting to contribution deletion, and ownership transfer or archival before owner closure.
   - Confirmed: narrow revocation cleanup after peer revocation; a restore never serves state that later deletions or restrictions forbid (journal design deferred to the Milestone 6 recovery ticket).
3. Federation
   - Confirmed: custom protocol for our own instances; other protocols deferred; server approval plus per-community opt-in; remote deletion cannot be guaranteed.
   - Confirmed: HTTPS plus standard HTTP Message Signatures, approved peer keys, stable operation identifiers, deduplication, host revisions, and pending/published/failed action states.
   - Ticket-level details: signature profile/library verification, key lifecycle, bounded retry/lease policy, protocol fields, reconciliation, and abuse limits.
4. Technology and architecture
   - Confirmed: Go backend; delegated selection of React with TypeScript frontend; PostgreSQL; modular application with in-process delivery worker and transactional outbox; maintainability and testability as priorities.
   - Confirmed: delegated frontend selections (Vite, React Router, TanStack Query, CSS Modules), client rendering, public link metadata, and internal-network-first search/discovery.
   - Confirmed: explicit SQL through pgx/sqlc, versioned migrations, REST/OpenAPI with generated frontend types, processed media on persistent storage, Compose/Caddy, and off-VPS daily encrypted backups.
   - Ticket-level details: migrations, module interfaces, API operations, dependency versions, sizing, queue settings, telemetry, and deployment configuration.
5. Design language
   - Confirmed: warm, restrained neighborhood clubhouse aesthetic with forum readability.
   - Confirmed: Clubhouse layout. Visual tokens are finalized in the first UI ticket. Mobile usability and keyboard navigation are required; fuller accessibility checks will be specified in UI tickets.

## Engineering approach

General design recommendations, followed by the accepted application structure:

- Organize behavior into modules with small interfaces. Keep authorization and federation rules in the backend; frontend components handle presentation and interaction.
- Test domain rules through module interfaces, persistence against a real database, frontend behavior through user interactions, and federation across independent instances.
- Within the accepted modular application, introduce separate infrastructure only when a concrete requirement justifies its operational cost.
- Keep protocol representations separate from domain meanings. Future protocol support may still require changes to identity, privacy, and delivery semantics.

### Accepted application structure

- A React/TypeScript frontend and one modular Go application per server, with PostgreSQL for relational data and durable delivery work.
- Modules: identity, communities and membership, discussions, moderation, and federation. Their interfaces should hide their rules and persistence details from callers.
- Write a domain change and its outgoing delivery records in the same database transaction. A background worker attempts delivery, records outcomes, and retries failures. This avoids losing a delivery intent when the process stops after committing a change.
- A background worker runs within the application initially, using PostgreSQL for durable delivery work. Processed media lives on persistent storage outside the application image.

### Accepted backend and API choices

- Go standard-library HTTP routing; pgx database driver and sqlc-generated query methods from explicit SQL; versioned SQL migrations. Migration tool and dependency versions are implementation-ticket choices, not architectural requirements.
- JSON REST browser API described by OpenAPI, with generated TypeScript request/response types. Browser and application share an origin; federation has a separate versioned interface.
- Federation over HTTPS with a documented RFC 9421 HTTP Message Signatures profile and a vetted implementation. No custom signing algorithm. Operator-approved peer keys establish trust; version, covered request components, replay rejection, rotation, and actor binding must be specified before protocol implementation.
- Selected test-tool defaults for ticket planning: Go tests and PostgreSQL integration tests; Vitest/Testing Library for frontend behavior; Playwright for key browser journeys; a two-server federation harness for failure cases. Exact versions and harness configuration belong to the foundation ticket.

### Search implementation

PostgreSQL full-text search covers permitted local/replicated content initially, avoiding a separate search service. Search snippets, counts, and results must not expose private content without current host authorization. Stale indexed content is not permission to serve it. Accepted as the initial search approach.

### Delivery design

- Recheck current authorization before sending or accepting work. Durable queues do not override revoked sharing or membership permissions.
- Use durable incoming identifiers to deduplicate retried operations. Claim work briefly, then perform network operations outside the database transaction. Lease expiry permits recovery after a worker crash.
- Treat remote commands separately from host-confirmed events. An author's home server must not confirm a publication merely because it queued a request.
- Version authoritative records so a delayed edit cannot resurrect deleted content or overwrite a newer state. Exact version fields, bounded retry policy, ordering, and reconciliation mechanics are implementation-ticket details to explain and test before their use.

## Draft schema — not migration-ready

These are proposed logical relations, not accepted column definitions or implementation. The community host owns authoritative content and permissions; replicas must retain that provenance. Local credentials never become federated account data.

| Area | Candidate relations | Purpose |
| --- | --- | --- |
| Identity | `accounts`, `local_credentials`, `sessions`, `account_tokens`, `registration_requests`, `server_invitations` | Represent local and known remote identities while keeping credentials, email, sessions, and verification/recovery tokens local. |
| Server trust | `servers`, `peer_connections`, `peer_keys` | Track remote identity, mutual connection approval, and HTTP Message Signature verification keys. Key lifecycle details belong to protocol design tickets. |
| Communities | `communities`, `community_peer_grants` | Store host, access preset, admission rule, owner, status, and explicit directional sharing approvals. |
| Participation | `memberships`, `membership_restrictions`, `community_invitations` | Track admission, roles, bans, and independently removable access restrictions. Revoked-peer suspension must not erase a ban or be lifted accidentally by re-peering. |
| Discussions | `discussions`, `replies` | Store host-authoritative contributions, author references, parent/thread relationships, revisions, and deletion placeholders. |
| Media | `attachments` | Store ownership, visibility association, type, size, and a reference to processed bytes on persistent storage. |
| Personal features | `bookmarks`, `notifications` | Store personal references and read state on the account's home server; notifications and bookmarks must respect current access and deletion state. |
| Moderation | `reports`, `report_evidence`, `moderation_actions` | Track reports, decisions, reasons, minimal audit metadata, and restricted evidence expiring at most 30 days after capture. |
| Federation | `incoming_operations`, `outgoing_events`, `deliveries`, `pending_commands` | Deduplicate requests, retain committed delivery intent, track attempts, and distinguish pending remote actions from host acceptance. Payload retention must honor deletion policy. |

### Core columns and constraints under review

Every relation has an appropriate primary key. Timestamps use timezone-aware values. Proposed identifiers use local UUID keys plus unique canonical URLs for federated entities; display handles are not foreign keys. Columns below are the important fields, not an exhaustive migration specification.

| Relation | Important fields and constraints |
| --- | --- |
| `accounts` | `id`, `home_server_id`, `canonical_url`, `username`, `display_name`, `avatar_id`, `status`; unique canonical URL and normalized `(home_server_id, username)`. |
| `local_credentials` | `account_id` primary/foreign key, normalized unique `email`, `email_verified_at`, `password_hash`; local accounts only; never serialized to peers. |
| `sessions` | `id`, `account_id`, unique `token_hash`, `created_at`, `expires_at`, `revoked_at`; raw session secrets are not stored. |
| `account_tokens` | `id`, `account_id`, `purpose`, unique `token_hash`, `expires_at`, `consumed_at`; purpose distinguishes email verification and password recovery. |
| `servers` | `id`, unique `canonical_origin`, `status`; the local server is explicitly identified. |
| `peer_connections` | `peer_server_id`, `local_approved_at`, `remote_approved_at`, `revoked_at`; effective peering requires both approvals and no revocation. |
| `communities` | `id`, `canonical_url`, `host_server_id`, `slug`, `name`, `description`, `owner_account_id`, `access_preset`, `admission_policy`, `preset_locked_at`, `status`; unique `(host_server_id, slug)` and valid preset/admission combinations. |
| `community_peer_grants` | `community_id`, `peer_server_id`, `status`, `generation`, approval/revocation timestamps; unique pair. A generation separates old work from a later sharing approval. |
| `memberships` | `id`, `community_id`, `account_id`, `admission_status`, `role`, `joined_at`; unique account/community pair. Ownership is represented by the community owner reference, not a competing owner role flag. |
| `membership_restrictions` | `id`, `membership_id`, `kind`, `source_peer_id` where applicable, `created_at`, `released_at`, `reason`; peer revocation, bans, and other restrictions cannot silently clear one another. |
| `discussions` | `id`, `canonical_url`, `community_id`, `author_account_id`, `title`, `body_markdown`, `revision`, timestamps, `deleted_at`; deleted text is cleared while identifiers remain for replies and reconciliation. |
| `replies` | `id`, `canonical_url`, `discussion_id`, `parent_reply_id`, `author_account_id`, `body_markdown`, `revision`, timestamps, `deleted_at`; parent must belong to the same discussion; no cascading deletion of descendants. |
| `attachments` | `id`, `owner_account_id`, content association, `storage_key`, detected media type, byte count, dimensions, `status`; access is derived from the associated content, not the obscurity of its storage key. |
| `bookmarks` | `account_id`, `discussion_id`, `created_at`; unique pair, local to the account's home server. |
| `notifications` | `id`, `recipient_account_id`, `kind`, `source_event_id`, target reference, `created_at`, `read_at`; deduplicate by recipient and event; prefer references over stored private text snippets. |
| `report_evidence` | `id`, `report_id`, restricted snapshot reference, `captured_at`, `expires_at`; expiry no later than capture plus 30 days, independently enforced on reads and cleanup. |
| `incoming_operations` | sending server, operation ID, payload digest, outcome reference, timestamps; unique sender/operation pair; reuse of an ID with different content is rejected. |
| `outgoing_events` | `id`, entity reference, entity revision, event type, required payload/reference, `created_at`; created in the domain transaction; payload retention must follow deletion rules. |
| `deliveries` | `id`, `event_id`, `recipient_server_id`, relevant sharing generation, `status`, `attempt_count`, `next_attempt_at`, `lease_until`, last error class; unique event/recipient pair. |
| `pending_commands` | `id`, initiating account, destination host, idempotency identifier, command type, expected revision where relevant, status, confirmed result reference; queued is distinct from published. |

Application transactions enforce invariants that span relations; database constraints enforce local uniqueness, references, and valid states. Invitation, report, key-management, and audit fields will be completed against their exact workflows before writing migrations.

### Proposed invariants to test

- Canonical remote identity includes its issuing server; identical local identifiers on different servers never collide.
- Each account has at most one membership record per community, with independent restrictions rather than a single state that can accidentally erase a ban.
- Each reply belongs to one discussion; its parent, if any, belongs to that same discussion. Deleting a parent does not cascade-delete replies.
- A peer connection and a community sharing approval are separate records. Neither substitutes for required membership authorization.
- A retried command cannot produce a second contribution, notification, or moderation action. Event processing and its deduplication marker commit together.
- Content changes and outgoing delivery intent commit together. Authorization is checked again when work is attempted.
- Deletion handling covers replicas, queued payloads, previews, search results, notifications, and attachments, rather than only clearing the main text column. Backup expiry is a separate policy.
- A compliant peer must obtain current authorization from the community host before returning private content, including search snippets, cached attachments, and notification content. Host outage makes these reads unavailable; public cached content remains available.
- Community history is visible to current eligible members regardless of join time, but revoked sharing or a membership restriction denies that access.
- Once the first discussion locks a community's preset, no ordinary update can change it.
- A password reset consumes its token once and revokes existing sessions; verification/recovery tokens never appear in logs or federation payloads.
- Report evidence expires independently of report status; expiry is checked before reads as well as by cleanup jobs.

### Accepted revocation-cleanup and restore policies

1. After peer revocation, permit a narrow authenticated cleanup channel for deletion requests, deletion markers, and revocation notices. It cannot return private content, create/edit contributions, grant permissions, or reactivate a membership. Cleanup still requires a trusted identity and may be refused when keys or the peer itself are untrustworthy; there is no guaranteed remote erasure. This qualifies the earlier blanket rejection of subsequent community actions (see ADR 0003).
2. A restored server starts isolated. Before serving restored content, replay independently retained deletion/access changes, expire report evidence and stale sessions, and reconcile with authoritative peers. Unverifiable affected content remains unavailable rather than being served from an old snapshot. Daily backups alone cannot recover changes made after the snapshot; the recovery ticket must specify a minimal off-VPS deletion/revocation journal or equivalent and its failure behavior. The principle is accepted; the journal mechanism is designed in the Milestone 6 recovery ticket.

Fresh private-read authorization must bind the requesting account, peer, community, current restrictions, sharing generation, and specific content revision/deletion state. A generic membership check cannot authorize an obsolete cached body or attachment. Report evidence is a separate restricted copy with independent expiry, so it neither keeps deleted originals readable nor disappears inadvertently when original attachments are removed.

## Accepted interface direction

- A desktop navigation rail contains joined communities, discovery, saved discussions, and notifications. On narrow screens this becomes compact navigation above a single content column.
- Discussion lists emphasize titles, short excerpts, community identity, and reply context. No popularity leaderboard or trending feed.
- Warm paper surfaces, dark readable text, muted plum accents, clear focus indicators, and restrained separators. Product-specific light and dark tokens preserve the same hierarchy.
- Serif headings lend an editorial feel; system sans-serif text supports controls and long conversations. Typography and spacing can be adjusted after reviewing the proposed layouts.
- Community visibility and host identity are visible where they affect joining or posting decisions. Restricted/unavailable content receives a clear explanatory state.
- The Clubhouse layout (persistent navigation rail) was selected over a top-navigation Reading Room layout after reviewing planning mockups.

## Implementation baseline

The architecture, logical schema, visual direction, and milestone plan are the implementation baseline. Exact SQL migrations, signing-library validation, numerical rate limits and retry intervals, dependency pins, and API field definitions are reviewed within the relevant tickets. They are explicitly deferred engineering details, not assumed behavior.

Deployment prerequisites remain the GitHub repository destination, VPS provider/capacity and domains, email delivery configuration, and an off-VPS backup destination. These do not prevent local implementation planning. Infrastructure is not provisioned. The repository is https://github.com/AbuDubu/amethyst; work is tracked in its Issues and Milestones.

## Decision records

Capture agreed domain meanings in `docs/CONTEXT.md` when resolved. Record consequential architectural trade-offs in `docs/adr/` when accepted. Recommendations and unanswered interview questions are not accepted decisions.
