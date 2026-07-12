# Waaris MVP Security Review

## Executive Summary

The reviewed local MVP has no known critical asset-execution path: it stores operational metadata only, has no blockchain or cryptographic execution, and restores liveness from every non-active metadata state. This hardening pass closed the highest-impact application issues around trustee self-approval, mutation during verification, malformed JSON, unbounded network/server I/O, and unverified migration behavior.

Public production deployment is still blocked on deployment controls not present in the repository: managed secrets, TLS/ingress, distributed rate limiting, production mail delivery, monitoring, backup/restore evidence, and an external penetration test.

## Resolved Findings

### SEC-001

- Severity: High
- Location: `services/enrollment/internal/application/service.go` (trustee and authoring operations)
- Evidence: Trustee creation/update now compares contact email to the owner email and requires lifecycle `active` before policy/contact changes.
- Impact: Without these checks, an owner could self-approve or rewrite trustee policy after dormancy began.
- Fix: Added `ErrSelfTrustee` and `ErrWillNotEditable`, with service and HTTP tests.

### SEC-002

- Severity: Medium
- Location: `services/auth/internal/transport/httpapi/handler.go:225`, `services/enrollment/internal/transport/httpapi/handler.go:510`
- Evidence: JSON decoding now verifies EOF after one decoded object.
- Impact: Trailing JSON could create inconsistent request interpretation across components.
- Fix: Reject trailing payload data with a structured `400` error.

### SEC-003

- Severity: Medium
- Location: `services/auth/internal/transport/httpapi/errors.go:76`, `services/enrollment/internal/transport/httpapi/errors.go:88`
- Evidence: API middleware now bounds/normalizes correlation IDs, sets no-store/nosniff/frame/referrer headers, contains panics, and logs only request metadata.
- Impact: Arbitrary correlation values could pollute observability; absent response headers and panic handling weaken browser/API defense in depth.
- Fix: Added protected transport middleware and negative contract tests.

### SEC-004

- Severity: Medium
- Location: `services/auth/cmd/server/main.go:41`, `services/enrollment/cmd/server/main.go:46`, health-only service mains
- Evidence: All exposed Go servers now set read, write, idle, header, and header-read limits; auth/enrollment readiness checks PostgreSQL.
- Impact: Default timeout/header settings permit avoidable resource exhaustion and false readiness.
- Fix: Applied explicit server limits and dependency-aware `/readyz` handlers.

### SEC-005

- Severity: Medium
- Location: `services/enrollment/internal/application/smtp_notifier.go:23`
- Evidence: SMTP delivery uses a context-aware dial and connection deadline.
- Impact: A stalled Mailpit/relay could otherwise block lifecycle or request processing indefinitely.
- Fix: Added bounded SMTP I/O while retaining durable queue state.

### SEC-006

- Severity: Medium
- Location: `infra/migrations/000005_mvp_hardening.up.sql`, `.github/workflows/ci.yml:31`
- Evidence: Response actor foreign key uses `ON DELETE SET NULL`; lifecycle/verification lookup indexes were added; CI runs `govulncheck` and disposable PostgreSQL repository/migration tests.
- Impact: Trustee account deletion could erase response evidence, and missing indexes/CI database checks increase operational risk.
- Fix: Added migration hardening and CI gates.

## Remaining Findings

### SEC-007

- Severity: High
- Location: deployment boundary; no ingress/IaC implementation is present
- Evidence: No distributed rate limiting, WAF, TLS ingress, or managed secret integration exists in the repository.
- Impact: A public endpoint can face credential stuffing and denial-of-service despite application validation.
- Fix: Require ingress-level rate limits, TLS, network policy, managed secrets, and monitoring before public exposure.
- False positive notes: A managed platform may supply these controls; verify concrete policies before launch.

### SEC-008

- Severity: Medium
- Location: `services/enrollment/internal/application/smtp_notifier.go`
- Evidence: SMTP is intentionally configured for local Mailpit and lacks TLS/auth/retry/dead-letter semantics.
- Impact: It is unsuitable for production delivery or dependable notification escalation.
- Fix: Introduce a separately reviewed authenticated TLS provider adapter with retry/dead-letter metrics before public deployment.

### SEC-009

- Severity: Medium
- Location: `services/enrollment/internal/infrastructure/postgres/store.go:833` audit persistence
- Evidence: Audit rows are append-only by application behavior, but the schema does not enforce immutable database permissions or trigger policy.
- Impact: A compromised database runtime role could alter audit history.
- Fix: Use least-privilege roles that deny audit updates/deletes and define audited retention procedures.

## Validation

- Full Go workspace tests and `go vet` passed.
- Web format, lint, tests, and production build passed.
- Compose interpolation validation passed with CI-only placeholders.
- PostgreSQL migration/repository integration tests are configured in CI; local execution requires `ENROLLMENT_INTEGRATION_DATABASE_URL`.
