---
name: waaris-architecture
description: Review or change Waaris system architecture, service boundaries, state-machine behavior, data flows, technology choices, or architecture decisions. Use for design reviews, implementation plans, cross-component changes, and ADR updates in this repository.
---

# Waaris Architecture

Read `.context.md`, `README.md`, `PLAN.md`, `DECISIONS.md`, `API_SPEC.md`, and `DATABASE.md` before deciding.

1. Map the request to clients, services, data/event stores, trust actors, chain, and external integrations.
2. Preserve boundaries: no plaintext, complete keys, or sensitive PII on-chain; off-chain services coordinate rather than custody.
3. Validate every lifecycle transition against `Active → PendingVerification → GracePeriod → Settled`; retain liveness reset before settlement.
4. State trade-offs, operational failure modes, legal boundaries, and a phased rollout.
5. Add or supersede an ADR for durable decisions, then synchronize the living project documents.
