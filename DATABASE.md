# Database Design (Foundation Draft)

## Principles

PostgreSQL stores only metadata required to coordinate a will. Redis stores ephemeral heartbeat and workflow scheduling data. NATS/Kafka events are not a system of record. No vault plaintext, private keys, trustee shares, DEKs, biometric templates, raw OTPs, or raw proof payloads belong in this design.

## PostgreSQL tables

| Table | Key columns | Notes |
|---|---|---|
| `users` | `id`, lowercase `email`, `password_hash`, `display_name`, timestamps | Owned only by `services/auth`; no protocol, biometric, wallet, or DID data |
| `refresh_tokens` | `id`, `user_id`, `token_hash`, `expires_at`, `revoked_at`, `created_at` | Stores SHA-256 hash of an opaque token; cascades on account deletion |
| `digital_wills` | `id`, `user_id`, `status`, `current_version`, `dormancy_days`, `grace_days`, `policy_version`, `consent_accepted_at`, timestamps, `deleted_at` | One current active will aggregate per user; soft delete allows later recreation |
| `will_release_preferences` | `will_id`, `category`, `created_at` | Current normalized release categories for the active will |
| `will_versions` | `id`, `will_id`, `user_id`, `version`, `status`, timing policy, `policy_version`, `consent_accepted_at`, `created_at` | Immutable snapshot of every create/update |
| `will_version_release_preferences` | `will_version_id`, `category`, `created_at` | Snapshot of release categories for each version |
| `consent_records` | `id`, `will_id`, `will_version_id`, `user_id`, `policy_version`, `consent_type`, `accepted_at` | Append-only consent audit tied to each version |
| `trustees` | `id`, `will_id`, `user_id`, `name`, lowercase `email`, `relationship`, timestamps | Owner-managed trusted contacts; unique per will/email |
| `heartbeats` | `id`, `will_id`, `user_id`, `source`, `occurred_at`, `created_at` | Persisted authenticated liveness metadata; no device proof or biometric data |
| `verification_requests` | `id`, `will_id`, `user_id`, `threshold_required`, `status`, timestamps | One pending request per will; local majority verification only |
| `verification_responses` | `id`, `request_id`, `trustee_id`, nullable `actor_user_id`, `decision`, `responded_at` | Append-only trustee decisions; user deletion preserves the response record by nulling actor account reference |
| `notifications` | `id`, `will_id`, `user_id`, nullable `trustee_id`, recipient metadata, content, `status`, timestamps | Durable local email queue/history; content must remain operational and non-sensitive |
| `audit_events` | `id`, nullable `user_id`/`will_id`, actor/event/correlation/details/timestamp | Append-only application audit stream for auth and MVP lifecycle events |

## Constraints and indexes

- `digital_wills.status` and `will_versions.status` are constrained to `draft` or `published`.
- `digital_wills_active_user_uidx` enforces at most one non-deleted active will per user.
- `will_release_preferences` and `will_version_release_preferences` constrain categories to `financial`, `private`, or `community_shareable`.
- `will_versions` is unique on `(will_id, version)` and indexed by `(will_id, version desc)` for history reads.
- `consent_records` is indexed by `(will_id, accepted_at desc)` for audit retrieval.
- `digital_wills.lifecycle_state` is constrained to `active`, `pending_verification`, `grace_period`, or `ready_for_execution`; a partial expression index supports dormancy scans.
- `verification_requests` allows one pending request per will and indexes pending creation time for lifecycle work.
- `verification_responses` indexes `(request_id, trustee_id, responded_at desc)` so current per-trustee decisions can be calculated without full table scans.
- Trustee lookup is indexed by `(lower(email), will_id)` for pending-verification authorization.
- Notification queue consumption is indexed by `(status, queued_at)`.
- Foreign keys cascade to will metadata and consent/version rows on account deletion; the application soft-deletes only the active `digital_wills` row during a normal will delete.
- `verification_responses.actor_user_id` uses `ON DELETE SET NULL`, preventing an unrelated trustee account deletion from erasing an existing response.
- Use row-level authorization in application/service layer initially; evaluate PostgreSQL RLS before multi-tenant production.
- `users.email` is unique, lowercase, and bounded to 254 characters; display names are bounded to 100 characters. `refresh_tokens.token_hash` is unique and `refresh_tokens_active_user_idx` supports session lookup/expiry cleanup.

## Implemented enrollment schema rationale

- `digital_wills` is optimized for the current read path and owner-level uniqueness.
- `will_versions` preserves immutable snapshots so each update increments a version without overwriting history.
- `consent_records` is separate from the current/snapshot tables so future policy, legal, or publication consent types can be added without reshaping the main aggregate.
- `will_release_preferences` and `will_version_release_preferences` were added because category preferences are multi-valued and need both a current normalized form and a historical snapshot without denormalized arrays.
- Workflow tables are separate from `will_versions`: lifecycle state changes and heartbeats do not alter authoring-policy version history.
- Notification queue/audit writes share the same database but are deliberately not a distributed outbox. Extracting services requires an outbox/idempotency design.

## Redis keys (indicative)

- `heartbeat:last:{willId}`: verified timestamp, TTL aligned to policy evaluation; never raw signed payload.
- `rate:{principal}:{route}`: rate-limit counter.
- `workflow:lock:{willId}`: short-lived distributed lock for serialized transition evaluation.

## Lifecycle and retention

- Define retention schedules with legal counsel before production; minimize records and make deletion jobs auditable.
- Backups are encrypted, access-controlled, tested for restore, and must preserve the same data-classification restrictions.
- Immutable chain hashes cannot be erased; therefore they must never encode PII or reversible personal data.

## Migration approach

Use ordered, reviewed SQL migrations, one logical change per migration, reversible where safe. CI provisions a fresh database and applies every migration; integration tests exercise constraints and rollback behavior. Seed data must be synthetic. The Makefile deliberately has no automated down-migration command: destructive rollback requires an explicit, reviewed operator command.

## Last updated

2026-07-13 — MVP workflow added in migration `000004`; index and foreign-key hardening added in `000005`.
