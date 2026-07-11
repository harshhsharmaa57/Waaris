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
