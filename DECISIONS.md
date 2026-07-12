# Architecture Decision Log

Decisions are append-only. Supersede an entry rather than silently changing it.

## ADR-001 — README is the normative project specification

- Status: accepted (2026-07-11)
- Decision: implementation and planning must conform to `README.md`; it must be read completely before implementation changes.
- Consequence: conflicting future requirements require an explicit README and ADR update.

## ADR-002 — Deliver a non-custodial, metadata-only foundation first

- Status: accepted (2026-07-11)
- Decision: initial services store only operational metadata, commitments, hashes, and audit references; no sensitive vault plaintext, full keys, or trustee shares.
- Consequence: early MVP demonstrates safe lifecycle coordination, not asset custody or content recovery.

## ADR-003 — Advanced cryptography and blockchain execution are gated

- Status: accepted (2026-07-11)
- Decision: no implementation of MPC, threshold cryptography, ZK, crypto-shredding, DP, or chain execution begins until backend, clients, infrastructure, and test scaffolding complete their exit criteria.
- Consequence: roadmap risk is front-loaded into conventional, testable platform work.

## ADR-004 — State transitions require reversibility before settlement

- Status: accepted (2026-07-11)
- Decision: the MVP models `Active`, `PendingVerification`, and `GracePeriod`, with signed liveness able to return a will to `Active`; settlement remains unavailable initially.
- Consequence: inactivity cannot produce irreversible action in early releases.

## ADR-005 — Chain and provider choices remain open

- Status: proposed (2026-07-11)
- Decision: defer Polygon/Arbitrum versus Cosmos, provider selection, and integrations until documented Phase 3 discovery.
- Consequence: introduce adapter interfaces and avoid vendor-specific coupling in early services.

## ADR-006 — Use a Go multi-module workspace with a small shared platform module

- Status: accepted (2026-07-11)
- Decision: backend services are separate Go modules in `services/`, coordinated by `go.work`; `platform/` contains only infrastructure primitives shared by service shells.
- Consequence: services remain independently buildable and deployable, while duplicated non-domain concerns are minimized. Domain logic must stay within its owning service; `platform` cannot become a business-logic dependency hub.

## ADR-007 — Start services with health-only HTTP shells

- Status: accepted (2026-07-11)
- Decision: each initial Go service exposes only `/healthz` and `/readyz`, uses bounded HTTP header reads, and shuts down gracefully.
- Consequence: CI and Kubernetes/Docker integration can be validated without prematurely introducing identity, lifecycle, data, or execution behavior.

## ADR-008 — Use Compose-managed PostgreSQL migrations and disposable local dependencies

- Status: accepted (2026-07-11)
- Decision: Docker Compose runs PostgreSQL, Redis, Mailpit, a one-shot `migrate/migrate` migration runner, backend shells, and the web dashboard. The first migration creates only the `waaris` schema.
- Consequence: developers share a reproducible local topology and migrations remain reviewed SQL. Automated down migrations are intentionally blocked in the Makefile to prevent accidental destructive rollback. NATS and observability are deferred to the remaining Phase 0 work; no table or sensitive metadata is introduced yet.

## ADR-009 — Use React, TypeScript, Vite, and Tailwind for the initial dashboard

- Status: accepted (2026-07-11)
- Decision: the trustee/nominee dashboard is a strict TypeScript React application built by Vite with Tailwind integration, ESLint, Prettier, Vitest, and pinned direct dependencies.
- Consequence: the frontend has a fast, reproducible quality baseline without implementing domain workflows or retaining sensitive data.

## ADR-010 — Enforce quality at local and CI boundaries

- Status: accepted (2026-07-11)
- Decision: EditorConfig/Git attributes, pre-commit hooks, Make targets, and GitHub Actions validate formatting, linting, tests, builds, Compose syntax, and image builds.
- Consequence: contributors receive early feedback and CI is the source of release evidence. Docker builds still require a running daemon in local environments.

## ADR-011 — Isolate account authentication in its own service

- Status: accepted (2026-07-12)
- Decision: `services/auth` owns user accounts, credential verification, session lifecycle, and self-profile APIs; enrollment and the other protocol services receive only authenticated identity context later.
- Consequence: account passwords and session records do not leak into protocol-domain services, and future DID/VC identity work can be introduced behind an explicit boundary.

## ADR-012 — Use bcrypt and rotated opaque refresh tokens

- Status: accepted (2026-07-12)
- Decision: password hashes use bcrypt cost 12. Access tokens are HS256 JWTs with a 15-minute default lifetime. Refresh tokens are 32-byte random opaque values, stored only as SHA-256 hashes, rotated atomically on use, and expire after 30 days by default.
- Consequence: no password, raw refresh token, or token secret is persisted in application data or logs. A compromise of the refresh-token table does not permit direct session replay; a compromise of the JWT secret still requires incident rotation and invalidation procedures.

## ADR-013 — Keep Digital Will enrollment isolated from authentication and later execution workflows

- Status: accepted (2026-07-12)
- Decision: `services/enrollment` owns only the authenticated user's Digital Will metadata, consent capture, and version history. It validates Bearer access tokens using the shared JWT secret/issuer contract, but it does not depend on `services/auth` internals or own passwords, sessions, trustees, heartbeat state, notifications, or execution behavior.
- Consequence: the enrollment boundary stays narrow and testable. Future heartbeat, witness, trustee, DID, and execution capabilities can evolve behind separate services without coupling account state to will-domain behavior.

## ADR-014 — Model the will as a current aggregate plus append-only version and consent records

- Status: accepted (2026-07-12)
- Decision: store the active will in `waaris.digital_wills`, append every create/update to `waaris.will_versions`, and record every accepted policy version in `waaris.consent_records`. Normalize release-category selections into separate current and versioned preference tables rather than embedding mutable arrays in the main rows.
- Consequence: the API can read the current will cheaply while preserving an immutable history for audit and rollback analysis. Consent evidence stays queryable per version, and category preferences remain normalized for future reporting and validation.

## ADR-015 — Soft-delete the active will and require consent on every write

- Status: accepted (2026-07-12)
- Decision: `DELETE /api/v1/will` marks the current will deleted instead of removing historical rows. `POST` and `PUT` both require `policyVersionAccepted` and record a fresh consent event tied to the newly created will version.
- Consequence: users can recreate a new active will later without erasing historical version/consent evidence, while every material metadata change remains associated with an explicit accepted policy version. Account deletion still cascades through the user foreign key and removes the user's will data.

## ADR-016 — Keep Compose secret interpolation strict and satisfy it in CI with placeholders

- Status: accepted (2026-07-13)
- Decision: retain `:?` required-variable checks in `docker-compose.yml` for secrets such as `POSTGRES_PASSWORD` and `AUTH_JWT_SECRET`. GitHub Actions supplies non-secret placeholder values only within the CI container-validation job so `docker compose config --quiet` and `docker compose build` can evaluate the file.
- Consequence: local development and production still fail closed when required secrets are missing, while CI can validate Compose syntax and image definitions without access to real secrets or insecure defaults in the committed Compose file.
