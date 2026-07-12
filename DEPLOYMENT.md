# Deployment and Operations Plan

## Environments

| Environment | Purpose | Data rule |
|---|---|---|
| Local | Developer workflow via Docker Compose | Synthetic data only |
| CI | Repeatable test and scanning runs | Ephemeral synthetic data only |
| Development | Shared integration | No real user data |
| Staging/testnet | Release rehearsal and contract testing | Synthetic or explicitly authorized non-sensitive test data |
| Production | Approved service delivery | Requires all release/security/legal gates |

## Initial topology

1. Docker Compose runs PostgreSQL 17, Redis 7, Mailpit, a one-shot SQL migration runner, DB-backed auth/enrollment services, the remaining Go service shells, and the static web dashboard. Enrollment requires `SMTP_ADDR` (set to `mailpit:1025` in Compose), `SMTP_FROM`, and `LIFECYCLE_TICK_INTERVAL`; auth and enrollment both require the shared JWT secret locally.
2. Kubernetes deploys independently scalable Go services with readiness/liveness probes, resource requests/limits, network policies, and rolling-release controls.
3. Terraform provisions environment-scoped network, managed data services, Kubernetes, monitoring, IAM, and secrets integration. State must be encrypted and access controlled.
4. HashiCorp Vault (or an approved equivalent) holds operational service secrets only; it must never hold user vault keys or trustee shares.

## CI/CD pipeline

1. Validate formatting, types, lint, tests, migrations, API compatibility, documentation, and license headers.
2. Run dependency, container, IaC, and secret scans; produce SBOMs.
3. Build immutable, versioned artifacts; sign/provenance-attest them where the platform supports it.
4. Deploy automatically only to ephemeral/local development environments after passing gates; staging and production require approved promotion policies.
5. Run smoke tests and verify metrics/error budgets after deployment; automatically halt rollout on failed readiness or defined safety alerts.

## Observability and reliability

- Emit structured, redacted logs with correlation IDs.
- Export request, heartbeat, state-transition, queue, notification, and dependency metrics; do not expose sensitive labels.
- Trace cross-service flows with redacted attributes.
- Define SLOs before public use, especially for heartbeat ingestion, delayed-event processing, and liveness-override handling.
- Alert on missed job schedules, transition anomalies, signature-verification failures, queue growth, error rate, and unauthorized access attempts.
- Auth and enrollment expose `/healthz` for process liveness and `/readyz` for PostgreSQL dependency readiness. Servers use 15-second read, 30-second write, 60-second idle, and 16 KiB header limits.
- Local notification dispatch uses Mailpit only. A failed email remains recorded as `failed`; production requires retry/dead-letter handling and provider delivery telemetry before launch.

## Backup, recovery, and safe failure

- Encrypt backups, test restore procedures regularly, and restrict recovery access.
- Design all workflow messages and transition handlers to be idempotent.
- If a dependency is unavailable or evidence is ambiguous, pause advancement and notify operators; never settle/execute by timeout alone.
- Document incident response, manual safe-hold, credential revocation, and communication procedures before beta.

## Release gates

Foundation: reproducible local setup, CI green, health checks, baseline monitoring, migration/restore tests.

MVP: state-machine integration/e2e tests, liveness override tests, threat-model review, accessibility baseline, no sensitive content/key handling.

Cryptography/chain: independent reviews/audits, testnet rehearsal, incident drills, and approved legal/privacy posture.

Production: all applicable gates plus external security assessment, disaster-recovery exercise, operational ownership, and legal counsel approval.

## Required production configuration

- Terminate TLS at a trusted ingress and do not expose PostgreSQL, Redis, Mailpit, or service debug endpoints publicly.
- Store `DATABASE_URL` and `AUTH_JWT_SECRET` in managed secrets; keep the JWT secret at least 32 bytes and rotate it through a documented incident procedure.
- Replace Mailpit with an authenticated TLS SMTP/API provider and set `SMTP_ADDR`/`SMTP_FROM` through managed configuration.
- Set ingress rate limits for authentication and state-changing routes. Application-level distributed rate limiting is not implemented.
- Run the lifecycle worker in exactly one leader or introduce a durable distributed scheduler before scaling enrollment horizontally; the current database transactions prevent duplicate pending requests but do not provide job observability.
- Provision PostgreSQL backups, restore tests, least-privilege roles, audit-retention controls, metrics, traces, and alerting before any public deployment.

## Last updated

2026-07-13 — local MVP lifecycle, Mailpit queueing, readiness, timeout, and production release requirements documented.
