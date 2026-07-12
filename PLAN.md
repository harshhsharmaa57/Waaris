# Implementation Plan

## Purpose

This is the executable roadmap for the README specification. It intentionally starts with a constrained, reversible MVP and defers high-risk cryptographic and decentralized components until the foundation is proven.

## Component map

| Area | Components | Initial outcome |
|---|---|---|
| Clients | Mobile app, web dashboard, USSD/SMS adapter | Accessible enrollment, check-in, witness flows |
| Core backend | Enrollment, heartbeat, notification, witness coordination | Safe metadata lifecycle and non-cryptographic dormancy workflow |
| Platform | PostgreSQL, Redis, NATS, Docker Compose, Kubernetes, Terraform | Reproducible local and deployable environments |
| Assurance | Unit, integration, contract, end-to-end, security tests; CI; observability | Evidence that state transitions and safeguards work |
| Chain | Registry, dormancy state machine, attestation verifier, oracle adapter | Later testnet coordination ledger |
| Advanced privacy | SSS/MPC, threshold signing/decryption, ZK, crypto-shredding, DP/PII pipeline, IPFS/Filecoin | Later audited category-specific execution |

## Phased plan

### Phase 0 — Foundation and governance

1. Create the monorepo layout described in README, version/toolchain manifests, editor settings, and license validation.
2. Add Docker Compose for PostgreSQL, Redis, NATS, local object storage/mock dependencies, and a developer bootstrap command.
3. Add CI for formatting, linting, unit tests, dependency/security scanning, image builds, and documentation checks.
4. Define API conventions, database migrations, authentication/session boundaries, structured logging/redaction, metrics, tracing, and error handling.
5. Add test harnesses and fixtures before business features.

Exit criteria: a clean checkout can run quality checks and local dependencies; services can expose health endpoints; CI reports are actionable.

Status on 2026-07-13: the local MVP workflow and hardening baseline are implemented. Remaining Phase 0 work is focused on NATS, observability, deployment manifests/Terraform, mobile skeletons, distributed abuse controls, and broader release automation.

### Phase 1 — Safe local MVP

1. Implement enrollment metadata: current milestone delivered the safe subset only: single-owner Digital Will metadata, timing policy, draft/published state, release categories, version history, and consent records. Trustee/witness references, nominees, and any sensitive-content references remain deferred.
2. Implement authenticated heartbeat intake, persisted liveness state, lifecycle evaluation, and immutable redacted audit events. Completed for the local MVP; signed device proofs and Redis acceleration remain deferred.
3. Implement notification scheduling and escalation. Completed for local Mailpit email with a durable PostgreSQL queue; production providers remain deferred.
4. Implement a local, non-ZK trustee confirmation workflow that can only advance to a grace period. Completed with majority approval and append-only responses.
5. Implement liveness override and state-transition guards: `Active → PendingVerification → GracePeriod → ReadyForExecution`; no execution exists. Completed.
6. Build a minimal web portal and mobile check-in/enrollment path against the APIs. API journey is complete; interactive dashboard/mobile workflows remain a Phase 2 delivery.

Exit criteria: full local API demo exercises enrollment, trustee setup, missed heartbeat, verification, grace period, ready metadata, and liveness reset; no plaintext vault data or keys are handled.

### Phase 2 — Product hardening and accessibility

1. Add multi-witness policy and diversity metadata validation without claiming identity truth that cannot be verified.
2. Implement offline-first witness drafts/sync conflict handling and a pluggable SMS/USSD adapter using sandbox credentials only.
3. Add nominee evidence-package metadata generation (no financial transfer integration).
4. Expand UI accessibility, localization readiness, recovery paths, audit views, and operational runbooks.
5. Add load, chaos, integration, end-to-end, and security tests for heartbeat and notification paths.

Exit criteria: evidence shows safe degradation under connectivity and dependency failures; accessibility and observability baselines are met.

### Phase 3 — Chain coordination baseline

1. Decide EVM L2 versus Cosmos using documented cost, security, developer-experience, and governance criteria.
2. Implement the registry/state-machine contract without private data, deploy only to a local chain, and write exhaustive state/property tests.
3. Add an authenticated, replay-safe oracle adapter and an off-chain/on-chain reconciliation worker.
4. Introduce testnet only after independent contract review and operational rollback/incident procedures exist.

Exit criteria: local-chain transitions mirror backend state exactly and adversarial tests cover unauthorized, duplicate, and stale submissions.

### Phase 4 — Advanced cryptography and privacy

Prerequisites: Phases 0–3 complete, independent architecture/security review, formal key-lifecycle specification, legal/privacy review, and explicit product approval.

1. Prototype and independently test Shamir share generation/distribution and threshold ceremony interfaces; never use production secrets.
2. Define and audit cryptographic shredding key lifecycle per bucket.
3. Add ZK witness-attestation circuit/prover/verifier with benchmarked parameters and independent review.
4. Add PII scrubber, privacy-budget ledger, trustee review queue, and DP pipeline with measurable utility/re-identification tests.
5. Add IPFS/Filecoin publication only for explicitly opted-in, sanitized artifacts.

Exit criteria: every primitive has a written threat model, test vectors, external audit findings disposition, and incident/recovery runbook.

### Phase 5 — Pilot and production readiness

1. Complete legal-partner evidence workflow and jurisdiction-specific operating policies.
2. Run external security audit, privacy review, disaster recovery exercise, and accessibility validation.
3. Conduct limited testnet/pilot with synthetic or explicitly authorized non-sensitive data.
4. Define production release gates, monitoring SLOs, support/escalation, and governance.

## Planned implementation order

Foundation → enrollment/heartbeat → notifications/witness grace flow → clients/accessibility → tests/operations hardening → contract baseline → advanced privacy/cryptography.

## Last updated

2026-07-13 — local MVP workflow and hardening baseline completed; production operational controls are next.
