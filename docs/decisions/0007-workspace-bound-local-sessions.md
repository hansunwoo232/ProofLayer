# ADR-0007 — Use Workspace-Bound Local Sessions for the MVP

- **Date:** 2026-08-21
- **Status:** Accepted for the local MVP
- **Owner:** Founding team

## Context

The Day 30 browser could create a Test Job through a protected loopback request,
but it had no operator identity, logout, or workspace ownership boundary. A
publicly reachable identity provider is premature, while an unscoped local
password would create authorization debt before the first durable database.

## Decision

Add an opt-in, single-operator local authentication service. Store the
bootstrap password as an Argon2id verifier, represent the user as a principal
bound to one canonical workspace, and issue opaque browser session tokens whose
server-side registry stores only SHA-256 digests.

Use `HttpOnly`, `SameSite=Strict` cookies, exact-origin and double-submit CSRF
checks, 30-minute sliding idle expiry, eight-hour absolute expiry, CSRF rotation
at login, and immediate server-side revocation at logout. Keep Runner identity
and browser identity as independent security boundaries.

Local authentication is enabled only when
`PROOFLAYER_LOCAL_ADMIN_PASSWORD` is configured. The no-auth loopback mode stays
available solely to preserve the Day 30 isolated acceptance flow.

## Alternatives considered

- **Basic authentication:** rejected because it provides no explicit logout,
  revocation, or session lifetime.
- **JWT browser tokens:** rejected because self-contained tokens complicate
  immediate logout and add claim-validation surface with no local MVP benefit.
- **OIDC/SAML now:** deferred until customer and deployment requirements are
  validated.
- **Browser local storage:** rejected for credentials and bearer tokens because
  script-readable persistence expands the impact of a client-side defect.

## Consequences

- browser requests can be attributed to a workspace-bound operator;
- a captured server session registry does not reveal reusable raw tokens;
- restarts clear all users and sessions because persistence is intentionally
  absent;
- one local administrator is not a production identity system; and
- rate limiting, MFA, recovery, and enterprise federation remain later work.

## Validation

- password hashes round-trip and malformed/unbounded hashes fail closed;
- unknown email and wrong password return the same public error;
- login rotates CSRF and produces hardened cookies;
- an authenticated session can create a Test Job;
- logout revokes the server-side session; and
- replaying the revoked token returns `AUTHENTICATION_REQUIRED`.
