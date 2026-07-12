# Database Design (Foundation Draft)

## Principles

PostgreSQL stores only metadata required to coordinate a will. Redis stores ephemeral heartbeat and workflow scheduling data. NATS/Kafka events are not a system of record. No vault plaintext, private keys, trustee shares, DEKs, biometric templates, raw OTPs, or raw proof payloads belong in this design.

## PostgreSQL tables

| Table | Key columns | Notes |
|---|---|---|
| `wills` | `id`, `public_key_commitment`, `state`, `dormancy_days`, `grace_days`, `policy_version`, timestamps | One row per data principal's will metadata |
| `will_categories` | `will_id`, `category`, `enabled` | Enum: financial/private/community_shareable; no content pointers initially |
| `participants` | `id`, `will_id`, `role`, `did_reference`, `eligibility_status`, `created_at` | Role: trustee/witness/nominee; pseudonymous references where possible |
| `witness_requests` | `id`, `will_id`, `status`, `requested_at`, `expires_at` | Delivery identifiers belong in a protected adapter store, not this table |
| `attestations` | `id`, `will_id`, `participant_id`, `channel`, `proof_commitment`, `attested_at`, `status` | Store commitment/reference, not raw ZK proof in later phases |
| `heartbeats` | `id`, `will_id`, `signed_payload_hash`, `occurred_at`, `verified_at`, `key_version` | Retain minimal audit metadata only |
| `state_transitions` | `id`, `will_id`, `from_state`, `to_state`, `reason_code`, `actor_type`, `correlation_id`, `occurred_at` | Append-only audit trail |
| `idempotency_keys` | `scope`, `key`, `request_hash`, `response_reference`, `expires_at` | Prevent duplicate state changes |
| `outbox_events` | `id`, `aggregate_id`, `type`, `payload_reference`, `created_at`, `published_at` | Transactional outbox; payload must be redacted |
| `consent_records` | `id`, `will_id`, `policy_version`, `consent_type`, `recorded_at`, `proof_hash` | Tracks category and publication opt-in |
| `users` | `id`, lowercase `email`, `password_hash`, `display_name`, timestamps | Owned only by `services/auth`; no protocol, biometric, wallet, or DID data |
| `refresh_tokens` | `id`, `user_id`, `token_hash`, `expires_at`, `revoked_at`, `created_at` | Stores SHA-256 hash of an opaque token; cascades on account deletion |

## Constraints and indexes

- `wills.state` is a constrained enum: `active`, `pending_verification`, `grace_period`, `settled`.
- Foreign keys cascade only to non-audit operational data; retain redacted audit records according to approved policy.
- Unique `(will_id, role, did_reference)` on participants and unique `(scope, key)` on idempotency keys.
- Index state/time queries: `wills(state, updated_at)`, `heartbeats(will_id, occurred_at desc)`, `witness_requests(status, expires_at)`, `state_transitions(will_id, occurred_at desc)`.
- Use row-level authorization in application/service layer initially; evaluate PostgreSQL RLS before multi-tenant production.
- `users.email` is unique, lowercase, and bounded to 254 characters; display names are bounded to 100 characters. `refresh_tokens.token_hash` is unique and `refresh_tokens_active_user_idx` supports session lookup/expiry cleanup.

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

2026-07-12 — authentication users and hashed refresh-token schema added in migration `000002`.
