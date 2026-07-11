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

## Threats and initial mitigations

| Threat | Baseline mitigation | Validation |
|---|---|---|
| False dormancy/execution | Signed heartbeat, multi-stage workflow, grace override, no MVP settlement | State-machine integration and property tests |
| Replay or forged proof | Canonical signed payload, nonce, timestamp, public-key verification | Negative and fuzz tests |
| Account takeover | Key rotation/recovery policy, step-up checks, session controls | Auth threat model and penetration test |
| Trustee/witness collusion | Future threshold/diversity policies; no release capability in MVP | Policy tests; later external review |
| Metadata disclosure | Minimization, encryption in transit/at rest, RBAC, redacted logs | Access-control and log inspection tests |
| Supply-chain compromise | Pin dependencies, SBOM, scanner gates, signed CI artifacts | CI enforcement |
| Availability failure | Health checks, queues, idempotency, backups/restores, SLO alerts | Failure-injection and recovery drills |
| Privacy re-identification | Defer publication; future DP budget plus review and privacy evaluation | Specialist-reviewed DP tests |

## Incident requirements

1. Stop further state advancement, preserve redacted evidence, and revoke compromised operational credentials.
2. Assess whether any pending verification/grace workflows need manual safe hold.
3. Notify affected parties and authorities only under an approved legal incident procedure.
4. Publish remediation and update this threat model before re-enabling affected capability.

## Review gates

- Before public beta: application security review, privacy review, DR exercise, and accessibility review.
- Before contracts or cryptography: independent expert review and test vectors.
- Before production: external penetration test, smart-contract audit where applicable, legal counsel approval, and incident tabletop exercise.

## Last updated

2026-07-11 — initial baseline derived from README threat model.
