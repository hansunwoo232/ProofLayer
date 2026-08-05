# Runner Identity Model v0.1

**Status:** Frozen for the technical MVP  
**Date:** August 5, 2026

## Identity

Each installed Runner has one opaque `runner_id`, one asymmetric identity key,
and immutable bindings to one `environment_id` and one `host_id`. All three IDs
are UUIDs and contain no hostname, username, tenant name, or customer content.

The persisted public identity record contains:

- Schema version
- Runner, environment, and host IDs
- Identity key ID, never the private key
- Registration timestamp
- State: `active` or `revoked`

## Registration

1. An authorized operator creates a single-use, short-lived enrollment grant.
2. The Runner generates its identity key locally.
3. The Runner submits its public key and grant over authenticated outbound TLS.
4. The Control Plane returns the immutable environment and host bindings.
5. The Runner validates and persists the identity record and private key using
   the operating-system protected key store.

Enrollment tokens and private keys must never enter logs, reports, scenario
parameters, or the central minimum-metadata result.

## Authorization

A job is eligible only when the identity is active and both job bindings exactly
match the local environment and host IDs. Revocation, missing identity, a
binding mismatch, or an invalid identity record fails closed before scenario
resolution.

Key rotation and remote registration transport are outside the Day 15 skeleton;
the model reserves them without weakening the current boundary.
