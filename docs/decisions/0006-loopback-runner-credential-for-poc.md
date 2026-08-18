# ADR-0006 — Use an Identity-Bound Credential for the Loopback Runner PoC

- **Date:** 2026-08-18
- **Status:** Accepted for the local PoC only
- **Owner:** Founding team

## Context

Day 28 implemented the Test Job lifecycle as an in-process boundary. The next
step needs a real outbound HTTP client without publishing unauthenticated
mutation routes or weakening the production mTLS direction. The current
Control Plane binds only to loopback and has no certificate authority yet.

## Decision

For the same-host loopback PoC, use one high-entropy bearer credential bound in
Control Plane configuration to one canonical Runner, environment, and host.
Keep Runner routes disabled when the credential is absent. Compare the token in
constant time before leasing or looking up a job.

The Runner client may use plain HTTP only when the target hostname is an IP
loopback address. Non-loopback targets require HTTPS. Every leased Test Job must
also pass pinned Ed25519 signature, identity, expiry, allowlist, parameter, and
nonce-replay checks locally.

## Consequences

- The browser cannot mutate the Runner lifecycle.
- A missing, wrong, or cross-Runner credential cannot consume a lease.
- The bearer token is never accepted over plain HTTP outside loopback and is
  never logged.
- The local token does not replace mTLS, certificate registration, revocation,
  or rotation.
- The Windows VM cannot use this HTTP-only loopback mode; its authenticated
  boundary is subsequent work.
- Nonce replay state is currently in memory and is lost on Runner restart.

## Validation

- Unauthorized lease attempts leave queue depth unchanged.
- Correct credentials lease exactly one host-bound signed job.
- Strict acknowledgement, stage, and completion requests drive the ordered
  lifecycle.
- Tampered, mismatched, expired, and replayed jobs fail locally.
- Client configuration rejects plaintext non-loopback Control Plane URLs.
