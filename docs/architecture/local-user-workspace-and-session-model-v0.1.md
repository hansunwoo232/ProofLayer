# Local User, Workspace, and Session Model v0.1

## Purpose

Define the smallest local authentication boundary required by the MVP without
turning the Day 30 loopback lab into a network identity system. The model is
single-workspace and single-operator today, but every authenticated principal
is explicitly bound to a workspace so later persistence does not require a
breaking authorization change.

## Data model

### Workspace

- `id`: canonical UUID
- `slug`: stable lowercase workspace key
- `name`: human-readable workspace name
- `created_at`: UTC creation time

### User

- `id`: canonical UUID
- `workspace_id`: mandatory owning workspace
- `email`: normalized lowercase login identifier
- `display_name`: operator-facing name
- `role`: `admin` for the local MVP
- `status`: `active`
- `password_hash`: server-only Argon2id PHC string; never serialized

### Session

- random session ID for internal audit correlation
- authenticated principal snapshot
- SHA-256 digest of the random bearer token
- creation and last-seen timestamps
- sliding idle expiry and fixed absolute expiry

The raw session token exists only in the browser cookie and request. The
in-memory session registry stores only its digest.

## Password storage

The bootstrap password is accepted only through
`PROOFLAYER_LOCAL_ADMIN_PASSWORD`. It is removed from the process environment
after the local authentication service is created. Passwords must contain
12–128 UTF-8 bytes.

The stored verifier uses Argon2id with:

- 64 MiB memory;
- three iterations;
- parallelism of two;
- a random 16-byte salt; and
- a 32-byte derived key.

Verification strictly parses and bounds every PHC parameter before allocating
memory. Login failures return one generic error for unknown email and wrong
password.

## Browser session rules

- `prooflayer_session`: random 256-bit value, `HttpOnly`, `SameSite=Strict`,
  path `/`; `Secure` is set when TLS is active.
- `prooflayer_csrf`: independently random value with the same browser
  protections and a 30-minute lifetime.
- login rotates the CSRF value before returning an authenticated session;
- mutation routes require exact loopback `Origin`, same-origin fetch metadata,
  and a constant-time double-submit CSRF match;
- sessions expire after 30 minutes idle or eight hours absolute, whichever is
  earlier; and
- logout revokes the server-side digest before clearing both cookies.

## Authorization boundary

Browser job creation and status reads require a valid authenticated session
when local authentication is enabled. Runner transport continues to use its
separate, host-bound credential and is not authorized by browser sessions.
The workspace binding is carried in the principal and returned to the browser,
but no client-supplied workspace identifier is trusted for authorization.

## Local MVP limitations

- user, password verifier, and sessions are in memory and disappear on restart;
- there is one configured administrator and one workspace;
- password reset, invitations, MFA, OIDC, SAML, durable revocation, and
  multi-workspace membership are intentionally deferred; and
- the service remains bound to `127.0.0.1` and must not be exposed to a LAN or
  public network.

## Validation

Automated tests cover password round trips and malformed hashes, generic
credential failures, workspace-bound principals, digest-only session storage,
idle and absolute expiry, CSRF rotation, authorized job creation, logout, and
rejection of a revoked session.
