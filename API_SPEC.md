# API Specification (Foundation Draft)

## Scope and conventions

This is the initial REST edge API; internal services use gRPC once service boundaries exist. External GraphQL is deferred until client query needs justify it. All endpoints are versioned under `/v1`, use JSON, TLS, UTC ISO-8601 timestamps, and opaque UUID identifiers. Future protocol state changes require authentication, authorization, an idempotency key, and a correlation ID. Authentication session endpoints use credential validation, unique account constraints, and one-time refresh-token rotation instead of caller-supplied idempotency keys.

`X-Correlation-Id` is accepted/generated for every request. `Idempotency-Key` is required for `POST` state changes. Error bodies use `{ "code", "message", "correlationId" }` and never disclose sensitive verification details.

## Resource model

| Resource | Purpose | Sensitive fields excluded |
|---|---|---|
| Will | Metadata, policy, state, public-key commitment | Vault content, private keys, shares |
| Participant | Role-bound DID/reference and eligibility metadata | Unnecessary identity/profile data |
| Heartbeat | Signed liveness event reference | Biometric data, signing key |
| Attestation | Witness workflow status and proof reference | Witness identity on public chain, raw proof in logs |
| Evidence package | Metadata and integrity hash for nominees | Financial credentials or transfer authority |
| User | Self-managed account profile | Password hash, raw refresh token, JWT secret |
| Session | Access/refresh token pair | Password and persisted raw refresh token |

## Authentication and profile endpoints

| Method and path | Purpose | Authentication |
|---|---|---|
| `POST /v1/auth/register` | Create an account and issue a session | Public; valid email and 12–128 character password |
| `POST /v1/auth/login` | Verify credentials and issue a session | Public; invalid credentials receive a generic response |
| `POST /v1/auth/refresh` | Exchange one refresh token for a rotated session | Public; refresh token is single-use |
| `POST /v1/auth/logout` | Revoke a refresh token if present | Public; opaque refresh token supplied in body |
| `GET /v1/users/me` | Read current user's profile | Bearer access token |
| `PATCH /v1/users/me` | Update current user's display name | Bearer access token |
| `DELETE /v1/users/me` | Delete current user and cascading refresh tokens | Bearer access token |

Registration/login request body:

```json
{"email":"person@example.com","password":"at-least-12-characters","displayName":"Optional name"}
```

Refresh/logout request body: `{ "refreshToken": "opaque-value" }`. A session response contains `user`, `accessToken`, `refreshToken`, and `accessTokenExpiresAt`. Access tokens are supplied as `Authorization: Bearer <token>`.

All errors use `{ "code", "message", "correlationId" }`. Validation failures return `400`; duplicate registration returns `409`; malformed/expired/reused credentials return generic `401`; protected resources without a valid Bearer token return `401`.

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

No endpoint accepts raw asset data, biometric data, key shares, decrypted content, financial transfer instructions, or production ZK proofs in the foundation. DID/VC integration, password-reset/recovery, MFA, SMS/USSD, GraphQL, contract/oracle APIs, and every digital-will workflow require separate approved specifications.

## Last updated

2026-07-12 — authentication and self-profile endpoint contract added.
