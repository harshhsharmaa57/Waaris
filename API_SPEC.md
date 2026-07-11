# API Specification (Foundation Draft)

## Scope and conventions

This is the initial REST edge API; internal services use gRPC once service boundaries exist. External GraphQL is deferred until client query needs justify it. All endpoints are versioned under `/v1`, use JSON, TLS, UTC ISO-8601 timestamps, and opaque UUID identifiers. Requests that change state require authentication, authorization, an idempotency key, and a correlation ID.

`X-Correlation-Id` is accepted/generated for every request. `Idempotency-Key` is required for `POST` state changes. Error bodies use `{ "code", "message", "correlationId" }` and never disclose sensitive verification details.

## Resource model

| Resource | Purpose | Sensitive fields excluded |
|---|---|---|
| Will | Metadata, policy, state, public-key commitment | Vault content, private keys, shares |
| Participant | Role-bound DID/reference and eligibility metadata | Unnecessary identity/profile data |
| Heartbeat | Signed liveness event reference | Biometric data, signing key |
| Attestation | Witness workflow status and proof reference | Witness identity on public chain, raw proof in logs |
| Evidence package | Metadata and integrity hash for nominees | Financial credentials or transfer authority |

## Initial endpoints

| Method and path | Purpose | Required state/role |
|---|---|---|
| `POST /v1/wills` | Create metadata-only will | Authenticated data principal |
| `GET /v1/wills/{willId}` | Read permitted will metadata/state | Authorized participant |
| `PATCH /v1/wills/{willId}` | Update mutable policy/participants | Data principal; `Active` only |
| `POST /v1/wills/{willId}/heartbeats` | Submit signed heartbeat | Data principal/device credential |
| `POST /v1/wills/{willId}/liveness-proofs` | Reset pending/grace state with fresh proof | Data principal/device credential |
| `POST /v1/wills/{willId}/attestation-requests` | Open witness workflow after eligibility | System service |
| `POST /v1/attestations/{id}/confirmations` | Confirm in MVP non-ZK flow | Eligible witness |
| `GET /v1/wills/{willId}/audit-events` | Read redacted audit history | Authorized participant/auditor |
| `GET /healthz`, `GET /readyz` | Liveness/readiness | Unauthenticated, network-restricted in production |

## State transition contract

| From | Event | To | Guard |
|---|---|---|---|
| `Active` | Dormancy threshold evaluation | `PendingVerification` | Policy elapsed; no valid recent heartbeat |
| `PendingVerification` | Valid heartbeat/liveness proof | `Active` | Signature, freshness, authorization verified |
| `PendingVerification` | Required MVP witness confirmation | `GracePeriod` | Eligible witness; idempotent confirmation |
| `GracePeriod` | Valid liveness proof | `Active` | Signature, freshness, authorization verified |
| `GracePeriod` | Expiry | `Settled` | Not implemented in foundation/MVP; future multi-party prerequisites |

## Example creation request

```json
{
  "publicKeyCommitment": "sha256:...",
  "dormancyDays": 180,
  "graceDays": 30,
  "categories": ["financial", "private", "community_shareable"],
  "trusteeThreshold": 3,
  "trustees": [{"did": "did:example:trustee-1"}],
  "witnesses": [{"did": "did:example:witness-1"}],
  "nominees": [{"reference": "nominee-ref-1"}]
}
```

## Non-goals

No endpoint accepts raw asset data, biometric data, key shares, decrypted content, financial transfer instructions, or production ZK proofs in the foundation. Authentication mechanism, DID method, SMS/USSD protocol, GraphQL surface, and contract/oracle APIs require separate approved specifications.

## Last updated

2026-07-11 — initial foundation API draft.
