# Task Register

## Operating rule

Update this file, `PROGRESS.md`, and `.context.md whenever a task is completed, reprioritized, blocked, or materially re-scoped.

## Done

- [x] Read the README specification completely before changes.
- [x] Analyze the requested architecture and identify components.
- [x] Create the initial living documentation set.
- [x] Review relevant OpenAI skills and create the Waaris engineering workflow skill suite.

## Next: Phase 0 foundation

- [ ] Establish repository layout and toolchain manifests for Go, TypeScript, React Native, Solidity, and Python components.
- [ ] Define a supported-version matrix and local developer bootstrap.
- [ ] Add Docker Compose for PostgreSQL, Redis, NATS, and service health checks.
- [ ] Add database migration framework and a metadata-only schema baseline.
- [ ] Add Go service template with health/readiness endpoints, config validation, logging, metrics, and tracing hooks.
- [ ] Add web-dashboard and mobile skeletons with lint/type/test scripts.
- [ ] Add CI quality gates, dependency scanning, secret scanning, and container build checks.
- [ ] Add test harnesses, fixtures, coverage reporting, and end-to-end test topology.
- [ ] Add local observability stack and operational runbooks.

## Phase 1 safe MVP

- [ ] Implement enrollment metadata and consent/version records.
- [ ] Implement signed heartbeat verification and replay protection.
- [ ] Implement notification/escalation workflow with test adapters.
- [ ] Implement non-ZK single-witness confirmation and grace-period transitions.
- [ ] Implement liveness override and audited state reset.
- [ ] Implement minimal enrollment/check-in and trustee portal user journeys.

## Explicitly deferred

- [ ] Smart-contract deployment and oracle integration — after foundation and safe MVP stability.
- [ ] Shamir/MPC, FROST/BLS, ZK proofs, and crypto-shredding — after backend/frontend/infrastructure/testing scaffolding and specialist review.
- [ ] Differential privacy, PII NLP, IPFS/Filecoin publication — after prerequisites, review, and explicit opt-in controls.
- [ ] Any production banking, UPI, registry, or telephony integration — after legal, vendor, and security approval.

## Last updated

2026-07-11 — skills selection and custom-skill task completed.
