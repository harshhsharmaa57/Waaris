# API Specification (Foundation Draft)

## Scope and conventions

This is the initial REST edge API; internal services use gRPC once service boundaries exist. External GraphQL is deferred until client query needs justify it. Auth endpoints currently use `/v1`; the enrollment workflow requested in Milestone 3 uses `/api/v1`. All endpoints use JSON, TLS, UTC ISO-8601 timestamps, and opaque UUID identifiers. Future protocol state changes require authentication, authorization, an idempotency key, and a correlation ID. Authentication session endpoints use credential validation, unique account constraints, and one-time refresh-token rotation instead of caller-supplied idempotency keys.

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
| Digital Will | Metadata-only lifecycle policy and release preferences | Asset plaintext, keys, trustee shares, nominees, witness identity |

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

## Digital Will enrollment endpoints

| Method and path | Purpose | Authentication |
|---|---|---|
| `POST /api/v1/will` | Create the authenticated user's active Digital Will | Bearer access token |
| `GET /api/v1/will` | Read the authenticated user's active Digital Will | Bearer access token |
| `PUT /api/v1/will` | Replace mutable Digital Will metadata and append a new version | Bearer access token |
| `DELETE /api/v1/will` | Soft-delete the authenticated user's active will | Bearer access token |
| `GET /api/v1/will/history` | Read immutable version history for the active will | Bearer access token |
| `GET /healthz`, `GET /readyz` | Liveness/readiness | Unauthenticated, network-restricted in production |

Create/update request body:

```json
{
  "state": "draft",
  "dormancyPeriodDays": 180,
  "gracePeriodDays": 30,
  "policyVersionAccepted": "2026-07",
  "releaseCategories": ["financial", "private"]
}
```

Rules:

- `state` must be `draft` or `published`.
- `dormancyPeriodDays` must be between 1 and 3650.
- `gracePeriodDays` must be between 1 and 365.
- `policyVersionAccepted` is required on every create/update and must be at most 64 characters.
- `releaseCategories` must contain one or more unique values from `financial`, `private`, and `community_shareable`.
- Each user can have only one non-deleted active will; a second `POST` returns `409`.

Read response shape:

```json
{
  "id": "uuid",
  "userId": "uuid",
  "state": "draft",
  "version": 1,
  "dormancyPeriodDays": 180,
  "gracePeriodDays": 30,
  "policyVersionAccepted": "2026-07",
  "consentAcceptedAt": "2026-07-12T18:30:00Z",
  "releaseCategories": ["financial", "private"],
  "createdAt": "2026-07-12T18:30:00Z",
  "updatedAt": "2026-07-12T18:30:00Z"
}
```

History response shape:

```json
{
  "history": [
    {
      "id": "uuid",
      "willId": "uuid",
      "userId": "uuid",
      "version": 2,
      "state": "published",
      "dormancyPeriodDays": 365,
      "gracePeriodDays": 45,
      "policyVersionAccepted": "2026-08",
      "consentAcceptedAt": "2026-07-12T18:45:00Z",
      "releaseCategories": ["community_shareable"],
      "createdAt": "2026-07-12T18:45:00Z"
    }
  ]
}

```

Errors:

- Validation failures return `400`.
- Missing/invalid Bearer tokens return `401`.
- Creating a second active will returns `409`.
- Missing active will returns `404`.

## State transition contract

| From | Event | To | Guard |
|---|---|---|---|
| `Active` | Dormancy threshold evaluation | `PendingVerification` | Policy elapsed; no valid recent heartbeat |
| `PendingVerification` | Valid heartbeat/liveness proof | `Active` | Signature, freshness, authorization verified |
| `PendingVerification` | Required MVP witness confirmation | `GracePeriod` | Eligible witness; idempotent confirmation |
| `GracePeriod` | Valid liveness proof | `Active` | Signature, freshness, authorization verified |
| `GracePeriod` | Expiry | `Settled` | Not implemented in foundation/MVP; future multi-party prerequisites |

Milestone 3 intentionally does not expose trustees, witnesses, nominees, heartbeats, liveness proofs, attestations, or execution endpoints yet.

## Non-goals

No endpoint accepts raw asset data, biometric data, key shares, decrypted content, financial transfer instructions, or production ZK proofs in the foundation. DID/VC integration, password-reset/recovery, MFA, SMS/USSD, GraphQL, contract/oracle APIs, trustee/witness flows, and every execution workflow require separate approved specifications.

## Last updated

2026-07-12 — Digital Will enrollment endpoint contract added.
