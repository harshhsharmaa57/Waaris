# API Specification (Foundation Draft)

## Scope and conventions

This is the initial REST edge API; internal services use gRPC once service boundaries exist. External GraphQL is deferred until client query needs justify it. Auth endpoints currently use `/v1`; the enrollment workflow requested in Milestone 3 uses `/api/v1`. All endpoints use JSON, TLS, UTC ISO-8601 timestamps, and opaque UUID identifiers. Future protocol state changes require authentication, authorization, an idempotency key, and a correlation ID. Authentication session endpoints use credential validation, unique account constraints, and one-time refresh-token rotation instead of caller-supplied idempotency keys.

`X-Correlation-Id` is accepted/generated for every request; invalid values are replaced with a generated UUID. Error bodies use `{ "code", "message", "correlationId" }` and never disclose sensitive verification details. Request bodies are limited to 1 MiB, reject unknown fields, and must contain exactly one JSON object. `Idempotency-Key` is a planned distributed-workflow control and is not yet enforced by the local MVP.

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
| Trustee | Owner-managed verification contact | Credential, private key, biometric or identity-proof data |
| Notification | Local email delivery queue/history | Provider credentials and sensitive asset content |
| Audit event | Redacted account/workflow activity record | Passwords, tokens, full secrets, sensitive payloads |

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
  "lifecycleState": "active",
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

## Local MVP workflow endpoints

All routes below require a valid Bearer access token. Owner-scoped routes operate only on the caller's active will. Trustee response routes additionally require that the caller's authenticated email matches a configured trustee contact for that request.

| Method and path | Purpose | Success |
|---|---|---|
| `POST /api/v1/trustees` | Add a trustee `{name,email,relationship}` while lifecycle is `active` | `201` trustee |
| `GET /api/v1/trustees` | List the caller's trustees | `200` `{trustees: []}` |
| `PUT /api/v1/trustees/{trusteeId}` | Replace a caller-owned trustee | `200` trustee |
| `DELETE /api/v1/trustees/{trusteeId}` | Remove a caller-owned trustee | `204` |
| `POST /api/v1/heartbeats` | Persist an authenticated heartbeat and restore active lifecycle if applicable | `201` liveness status |
| `GET /api/v1/heartbeats` | Read current liveness status | `200` liveness status |
| `GET /api/v1/heartbeats/history` | Read heartbeat history | `200` `{history: []}` |
| `GET /api/v1/verifications/pending` | List pending requests assigned to the caller's email | `200` `{pending: []}` |
| `POST /api/v1/verifications/{requestId}/approve` | Append an approval response | `204` |
| `POST /api/v1/verifications/{requestId}/reject` | Append a rejection response | `204` |
| `POST /api/v1/verifications/{requestId}/abstain` | Append an abstention response | `204` |
| `GET /api/v1/notifications/history` | Read notification history for caller-owned will | `200` `{history: []}` |
| `GET /api/v1/audit/history` | Read account/workflow audit history | `200` `{history: []}` |

Trustee rules:

- Names and relationships are required and limited to 100 characters; email is required, normalized to lowercase, and limited to 254 characters.
- Duplicate trustee email per will returns `409`.
- The will owner cannot be a trustee (`400`).
- Published wills retain at least one trustee. Policy and trustee changes return `409` once lifecycle is no longer `active`.

Liveness status contains `willId`, `lifecycleState`, `lastHeartbeatAt`, `pendingVerificationStartedAt`, `gracePeriodStartedAt`, and `readyForExecutionAt`.

## State transition contract

| From | Event | To | Guard |
|---|---|---|---|
| `active` | Background lifecycle check after configured dormancy interval | `pending_verification` | Published will, at least one trustee, no later heartbeat |
| `pending_verification` | Authenticated heartbeat | `active` | Owner authentication; pending request is cancelled |
| `pending_verification` | Majority of current trustee responses approve | `grace_period` | `floor(trustee_count / 2) + 1` approvals; responses are append-only |
| `grace_period` | Authenticated heartbeat | `active` | Owner authentication; workflow is cancelled |
| `grace_period` | Grace interval expires | `ready_for_execution` | Metadata only; no notification, cryptography, transfer, or execution occurs |
| `ready_for_execution` | Authenticated heartbeat | `active` | Safety override because no irreversible execution exists in this MVP |

The lifecycle worker runs at startup and at `LIFECYCLE_TICK_INTERVAL` (default one minute). Mailpit email notifications are queued for dormancy, verification start, grace start, and liveness recovery.

## Non-goals

No endpoint accepts raw asset data, biometric data, device signatures, key shares, decrypted content, financial transfer instructions, or production ZK proofs. DID/VC integration, password-reset/recovery, MFA, SMS/USSD, GraphQL, contract/oracle APIs, production email providers, blockchain execution, and every asset-execution workflow require separate approved specifications.

## Last updated

2026-07-13 — local MVP trustee, heartbeat, verification, notification, and audit contract added; hardening behavior documented.
