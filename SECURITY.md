# Security Baseline and Threat Model

## Security posture

Waaris is safety-critical: false execution can expose private data or harm a living person. The initial release must therefore be coordination-only, reversible, and fail closed. This document is not a substitute for an external security assessment.

## Data handling rules

| Data class | Allowed persistence | Prohibited persistence |
|---|---|---|
| Operational metadata | PostgreSQL with access controls and retention policy | Unnecessary PII or raw content |
| Liveness proofs | Short-lived verification input plus redacted audit reference | Device biometric templates, raw credentials |
| Keys/shares | None in foundation services | Master keys, reconstructed keys, trustee shares, DEKs |
| Chain records | Hashes, commitments, state, proof references | Plaintext, PII, witness identity, ciphertext containing personal data |
| Logs/telemetry | Structured, redacted event IDs and outcomes | Tokens, secrets, proof payloads, contact details, message bodies |

## Required safety controls

- Verify signatures, enforce nonce/replay protection, expiry, authorization, and rate limits for every state-changing request.
- Keep an append-only, redacted audit trail with actor, action, policy version, correlation ID, and outcome.
- Require independent witness evidence and a grace period; a heartbeat lapse never executes a will.
- Use least privilege, service-to-service authentication, secret-manager references, encrypted transport, and separate environments.
- Make notification, registry, oracle, and telecom failures fail closed; retry idempotently and alert operators.
- Apply secure defaults: deny unknown states, validate all inputs, protect admin endpoints, and use dependency/secret scanning in CI.
- Conduct threat modeling and specialist review before each cryptographic, contract, identity, or external-integration milestone.
- Hash passwords with bcrypt cost 12; never log, return, or persist a plaintext password.
- Sign short-lived JWT access tokens with a 32-byte-or-longer secret from environment/secret management; rotate opaque refresh tokens and persist only their SHA-256 hashes.
- Record Digital Will updates as append-only versions plus append-only consent events; never overwrite version history or store sensitive asset payloads in enrollment tables.
- Lock policy/trustee changes outside `active`, reject self-trustee configuration, and require a majority of configured trustees before grace period metadata can begin.
- Treat `ready_for_execution` as reversible metadata while no execution exists; authenticated liveness always returns the local MVP to `active`.
- Bound HTTP headers, request bodies, JSON parsing, SMTP I/O, and server read/write/idle durations. Log correlation ID, method, path, status, and duration only.

## Threats and initial mitigations

| Threat | Baseline mitigation | Validation |
|---|---|---|
| False dormancy/execution | Signed heartbeat, multi-stage workflow, grace override, no MVP settlement | State-machine integration and property tests |
| Replay or forged proof | Canonical signed payload, nonce, timestamp, public-key verification | Negative and fuzz tests |
| Account takeover | Key rotation/recovery policy, step-up checks, session controls | Auth threat model and penetration test |
| Trustee/witness collusion | Future threshold/diversity policies; no release capability in MVP | Policy tests; later external review |
| Metadata disclosure | Minimization, encryption in transit/at rest, RBAC, redacted logs | Access-control and log inspection tests |
| Digital Will metadata tampering | Transactional current-row updates plus append-only `will_versions` and `consent_records`; correlation IDs on every request | Service/repository tests; future DB integration tests |
| Supply-chain compromise | Pin dependencies, SBOM, scanner gates, signed CI artifacts | CI enforcement |
| Availability failure | Health checks, queues, idempotency, backups/restores, SLO alerts | Failure-injection and recovery drills |
| Privacy re-identification | Defer publication; future DP budget plus review and privacy evaluation | Specialist-reviewed DP tests |
| Credential stuffing/token replay | Bcrypt verification, generic login errors, short-lived access tokens, hashed and rotated refresh tokens | Unit/HTTP flow tests; future rate limiting and monitoring |
| Owner self-approval or mid-workflow policy manipulation | Reject owner email as trustee; lock authoring/contact changes once lifecycle leaves `active` | Unit and HTTP authorization/state tests |
| SMTP dependency stall | Durable queue status plus bounded TCP/SMTP deadlines | Notification adapter tests; Mailpit integration rehearsal |
| Multi-replica lifecycle contention | PostgreSQL row transactions and unique pending request index | Repository integration test; production load test pending |

## Incident requirements

1. Stop further state advancement, preserve redacted evidence, and revoke compromised operational credentials.
2. Assess whether any pending verification/grace workflows need manual safe hold.
3. Notify affected parties and authorities only under an approved legal incident procedure.
4. Publish remediation and update this threat model before re-enabling affected capability.

## Review gates

- Before public beta: application security review, privacy review, DR exercise, and accessibility review.
- Before contracts or cryptography: independent expert review and test vectors.
- Before production: external penetration test, smart-contract audit where applicable, legal counsel approval, and incident tabletop exercise.

## Residual MVP risks

- JWTs use one shared HS256 secret across auth and enrollment. Rotate through managed secret storage on incident; move to asymmetric verification or introspection before multi-environment public launch.
- No distributed rate limiting, WAF, account recovery/MFA, or verified trustee identity exists. Enforce rate limits at ingress now and add identity/abuse controls only under a separately approved specification.
- Mailpit SMTP is local-development only. A production relay must use TLS, authenticated credentials, retry/dead-letter handling, delivery monitoring, and privacy review.
- Audit rows are append-only in application code, not protected by a database role/trigger policy. Production DB permissions must deny updates/deletes to the application runtime role except approved retention procedures.
- TLS, ingress network policy, backups/restore drills, metrics, tracing, and alerting are not included in the repository yet and remain release gates.

## Last updated

2026-07-13 — MVP authorization, transport, lifecycle, SMTP, and residual-risk controls reviewed.
