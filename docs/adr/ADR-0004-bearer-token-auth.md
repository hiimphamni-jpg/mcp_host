# ADR-00004: Bearer-Token Authentication for the Gateway

- **Status:** Proposed
- **Date:** 2026-08-07
- **Decision makers:** Architect (Flow 1 Design — FEAT-00008)
- **Related feature:** FEAT-00008 (HTTP/SSE API Gateway)

## Context

The gateway is a public-facing HTTP surface. It must be authenticated before
touching the LLM path or starting any MCP process. There is no user model, no
database, and no identity provider in the MVP. The auth mechanism must be cheap,
stateless, and testable, matching the minimal single-operator deployment of the
rest of the host.

## Decision

Require a static **Bearer token** delivered as `Authorization: Bearer <token>`,
configured via environment. Two variables are supported:

- `GATEWAY_API_TOKEN` — the primary token (required when the gateway runs).
- `GATEWAY_API_TOKENS` — comma-separated list, for **rotation**: the primary
  plus additional accepted values. Both may be set if you set `GATEWAY_API_TOKENS`.

Validation uses constant-time comparison (`crypto/subtle`) against every
configured token. A `request_id` is assigned before the auth check for
correlation. Failure returns HTTP `401` with the standard JSON error envelope
**before** any SSE; `GET /healthz` is unauthenticated.

Secrets policy from `rules/security.md` applies: tokens come only from
environment (never code, never logged). Tokens are treated as shared server
secrets; there is no per-user identity in MVP.

## Rationale

- An identity provider (OAuth2/OIDC), user accounts, and per-user JWTs are
  explicitly out of scope — no user model exists.
- A single static shared secret is the smallest authentication that closes the
  surface; constant-time comparison prevents timing side channels.
- The MVP operator (`rules/` least-privilege) issues the token to trusted
  callers; a browser frontend calls through a server-side proxy (or a Next.js
  route handler) that owns the header, so the token never ships to the client.

## Consequences

### Positive

- No DB, no sessions, no provider; set an env var and the gateway is behind auth.
- Trivial unit tests; no external service needed.
- Rotation via `GATEWAY_API_TOKENS` allows old+new during cutover.

### Negative

- Tokens are shared secrets; rotation currently requires an env change +
  restart (auto-reload is future work).
- No per-user authorization/rate attribution; all callers are equal (rate
  limiting is a follow-up hardening, see TDD §4).
- Browser clients cannot use SSE `EventSource` (no header support); the
  `fetch` + `ReadableStream` path must be used (ADR-00002).

### Neutral

- Relationship to other authentication layers (mutual TLS in front, etc.) is
  orthogonal and future.

## Alternatives Considered

1. **OAuth2 / OpenID Connect** — no identity provider or user model exists;
   heavy. Rejected for the MVP.
2. **Per-request signed JWTs** — needs a signing secret and a user/attribute
   source; no source exists. Rejected.
3. **Anonymous with IP allow-list** — weak and brittle; rejected.
4. **DB-backed API-key hashes** — no database in the MVP; rejected.

## References

- `rules/security.md` (auth, secrets)
- `docs/adr/ADR-0002-http-sse-transport.md`
- `docs/adr/ADR-0003-sse-event-contract.md`