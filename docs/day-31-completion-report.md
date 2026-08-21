# Day 31 Completion Report

**Date:** August 21, 2026

**Status:** Complete; the local MVP now has a workspace-bound operator session
and a defined information architecture.

## Gate

The Control Plane must authenticate one local operator without storing a
reusable password or session token in the browser or server registry, preserve
the Day 30 isolated acceptance mode, and make login and logout behavior
verifiable before durable user storage is introduced.

## Implemented

- Explicit Workspace, User, Principal, and Session domain models
- Mandatory workspace binding for the local administrator principal
- Argon2id password verifier with strictly bounded PHC parsing
- Generic credential failure behavior for unknown users and wrong passwords
- Random 256-bit browser session tokens with digest-only server storage
- Thirty-minute sliding idle expiry and eight-hour absolute expiry
- Immediate server-side session revocation on logout
- `HttpOnly`, `SameSite=Strict` session and CSRF cookies
- Exact-origin, fetch-metadata, and double-submit CSRF enforcement
- CSRF rotation after successful login
- Protected browser job creation and status routes when authentication is on
- Separate browser and Runner authentication boundaries
- English local sign-in surface and authenticated Sign out control
- Opt-in startup through `PROOFLAYER_LOCAL_ADMIN_PASSWORD`
- Automatic removal of the bootstrap password from the process environment
- Backward-compatible no-auth loopback mode for Day 30 reproduction
- MVP navigation, entity hierarchy, route direction, and future role model
- ADR-0007 for the local session decision

## Acceptance result

| Acceptance item | Result |
|---|---|
| Password is represented only by an Argon2id verifier | PASS |
| Malformed or unbounded password hashes fail closed | PASS |
| Unknown email and wrong password share one public error | PASS |
| Authenticated principal is bound to one workspace | PASS |
| Server registry stores only the session token digest | PASS |
| Idle and absolute session expiry are enforced | PASS |
| Login rotates CSRF and issues hardened cookies | PASS |
| Authenticated operator can create one Test Job | PASS |
| Logout revokes the server-side session immediately | PASS |
| Reusing a revoked session is rejected | PASS |
| Runner routes remain independent of browser sessions | PASS |
| Day 30 no-auth isolated mode remains available | PASS |

## Validation

- Full Control Plane unit and HTTP lifecycle test suite
- Argon2id round-trip, wrong-password, and malformed-hash tests
- Workspace-principal, digest-storage, idle-expiry, absolute-expiry, and logout
  tests
- Browser login, authenticated job creation, session document, and revoked
  token tests
- Control Plane and Runner Go race detector
- Control Plane and Runner static analysis
- Windows Arm64 Runner cross-compilation
- Versioned contract examples
- Authenticated dashboard and login-surface validator
- Repository whitespace validation

## Security boundary

This is still a loopback-only local MVP. User and session state are ephemeral.
It does not yet provide rate limiting, MFA, recovery, invitations, durable
revocation, multi-user roles, OIDC, or SAML. It must not be exposed to a LAN,
customer environment, or public network.

The full model and limits are documented in
[`docs/architecture/local-user-workspace-and-session-model-v0.1.md`](architecture/local-user-workspace-and-session-model-v0.1.md).
The product information architecture is documented in
[`docs/design/mvp-information-architecture-v0.1.md`](design/mvp-information-architecture-v0.1.md).

## Commercial work

Customer outreach remains deferred under ADR-0002, as requested by the
founder. Day 31 strengthens the product boundary without changing that
commercial sequencing decision.
