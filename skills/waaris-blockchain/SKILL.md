---
name: waaris-blockchain
description: Assess, design, implement, test, or audit Waaris Solidity contracts, oracle adapters, chain state, ZK attestation interfaces, and testnet workflows. Use only after foundational backend, frontend, infrastructure, and testing scaffolding exit criteria are complete.
---

# Waaris Blockchain

Read `.context.md`, `PLAN.md`, `DECISIONS.md`, and `SECURITY.md`. If the prerequisite gate is incomplete, document the request and stop rather than implementing it.

1. Confirm chain selection, threat model, specialist review, and local test strategy are approved.
2. Store only commitments, hashes, state, and approved proof references; reject PII, plaintext, key material, and raw sensitive ciphertext.
3. Model and exhaustively test authorization, state transitions, replay/duplicate protection, timestamp boundaries, oracle failures, and liveness resets.
4. Keep off-chain and on-chain state reconciliation explicit, idempotent, and observable.
5. Require independent contract/cryptography review before testnet and update ADR, API, security, deployment, progress, and task records.
