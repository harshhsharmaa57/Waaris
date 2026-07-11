---
name: waaris-security
description: Threat-model, review, or implement Waaris security-sensitive behavior involving identity, signatures, liveness, witnesses, trustees, encryption, privacy, secrets, data retention, external providers, contracts, or production access. Use before any security-relevant design or code change.
---

# Waaris Security

Read `SECURITY.md`, `.context.md`, and the affected API/database/deployment documents. Use the installed `security-threat-model` and `security-best-practices` skills for general methods.

1. Identify assets, trust boundaries, attackers, abuse cases, and failure modes; prefer fail-closed behavior.
2. Verify data minimization: do not persist plaintext vault content, biometric templates, master keys, shares, DEKs, OTPs, raw proofs, or secrets in logs.
3. Require authorization, freshness, replay protection, idempotency, auditability, rate limits, and redaction on state-changing flows.
4. Confirm inactivity alone cannot settle a will; require independent verification and grace-period liveness override.
5. Add adversarial tests and update `SECURITY.md`; stop for specialist/legal review at crypto, contract, privacy, or real-provider gates.
