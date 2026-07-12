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

1. Docker Compose now runs PostgreSQL 17, Redis 7, Mailpit, a one-shot SQL migration runner, the DB-backed auth and enrollment services, the remaining Go service shells, and the static web dashboard. Enrollment and auth both require the shared JWT secret locally. NATS and observability join in the remaining Phase 0 work.
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

## Last updated

2026-07-12 — Milestone 3 added DB-backed enrollment runtime requirements to the local topology.
