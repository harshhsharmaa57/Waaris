# Task Register

## Operating rule

Update this file, `PROGRESS.md`, and `.context.md whenever a task is completed, reprioritized, blocked, or materially re-scoped.

## Done

- [x] Read the README specification completely before changes.
- [x] Analyze the requested architecture and identify components.
- [x] Create the initial living documentation set.
- [x] Review relevant OpenAI skills and create the Waaris engineering workflow skill suite.

## Done: Milestone 1 — Project Foundation

- [x] Create the monorepo skeleton and placeholders for all README architectural boundaries.
- [x] Create Go workspace modules and independently buildable health-only shells for enrollment, heartbeat, notification, and witness coordination.
- [x] Create the React + TypeScript + Tailwind web-dashboard scaffold with lint, test, format, and production-build scripts.
- [x] Add Docker Compose for PostgreSQL, Redis, Mailpit, migrations, backend shells, and frontend shell.
- [x] Add PostgreSQL migration baseline and migration runner.
- [x] Add `.env.example` files, Makefile commands, EditorConfig, Git attributes/ignore rules, and pre-commit configuration.
- [x] Add GitHub Actions checks for Go formatting/tests/vet/build, web format/lint/test/build, and Compose/image validation.

## Done: Milestone 2 — Authentication and User Management

- [x] Implement isolated `auth` service with clean domain/application/infrastructure/HTTP boundaries.
- [x] Add registration, login, access-token refresh rotation, and logout endpoints.
- [x] Add bcrypt password hashing and HS256 JWT authentication middleware.
- [x] Add authenticated self-profile retrieval, update, and deletion endpoints.
- [x] Add `users` and hashed `refresh_tokens` migrations, repository implementations, request validation, and structured errors.
- [x] Add application unit tests, HTTP integration-flow tests, and opt-in PostgreSQL integration test.
- [x] Update local Compose, environment examples, Makefile/CI package checks, database, API, security, and decision documentation.

## Done: Milestone 3 — Digital Will Enrollment

- [x] Implement isolated `enrollment` service with clean domain/application/infrastructure/HTTP boundaries.
- [x] Add authenticated CRUD endpoints for one active Digital Will per user plus version-history retrieval.
- [x] Support `draft` and `published` states, version increments on every write, timestamps, dormancy/grace policy storage, policy-version acceptance, and normalized release-category preferences.
- [x] Add append-only consent records and append-only will-version snapshots while keeping the current aggregate separately queryable.
- [x] Add PostgreSQL migrations for `digital_wills`, `will_versions`, `consent_records`, and normalized release-preference tables.
- [x] Add application tests, HTTP integration/contract tests, and opt-in PostgreSQL migration/repository integration tests.
- [x] Update local Compose plus the affected context, plan, progress, decision, database, API, deployment, and security documentation.

## Remaining Phase 0 foundation

- [ ] Establish repository layout and toolchain manifests for Go, TypeScript, React Native, Solidity, and Python components.
- [ ] Define a supported-version matrix and local developer bootstrap.
- [ ] Add Docker Compose for PostgreSQL, Redis, NATS, and service health checks.
- [ ] Add database migration framework and a metadata-only schema baseline.
- [ ] Add Go service template with health/readiness endpoints, config validation, logging, metrics, and tracing hooks.
- [ ] Add web-dashboard and mobile skeletons with lint/type/test scripts.
- [ ] Add CI quality gates, dependency scanning, secret scanning, and container build checks.
- [ ] Add test harnesses, fixtures, coverage reporting, and end-to-end test topology.
- [ ] Add local observability stack and operational runbooks.

## Next: Phase 1 safe MVP

- [x] Implement enrollment metadata and consent/version records.
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

2026-07-12 — Milestone 3 completed; heartbeat is the next recommended feature milestone.
