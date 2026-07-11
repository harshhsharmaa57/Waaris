---
name: waaris-backend
description: Design, implement, test, debug, refactor, or review Waaris Go services, REST/gRPC APIs, PostgreSQL/Redis/NATS integration, React client contracts, database migrations, and dependency updates. Use for all foundation and safe-MVP application work.
---

# Waaris Backend

Read `.context.md`, `API_SPEC.md`, `DATABASE.md`, `SECURITY.md`, and affected ADRs first.

1. Make one vertical, reviewable change; keep Go services metadata-only and expose health/readiness endpoints.
2. Enforce typed validation, authz, signature freshness/replay controls where relevant, idempotency, transactional outbox events, and structured redacted logs.
3. Use ordered migrations with constraints/indexes and synthetic fixtures; never add sensitive vault/key/proof fields.
4. Test units, persistence, API contracts, and state transitions, including duplicate, stale, unauthorized, and dependency-failure paths.
5. Update API/database/security/progress/task docs and commit only verified changes.
