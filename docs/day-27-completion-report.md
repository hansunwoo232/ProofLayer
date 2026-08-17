# Day 27 Completion Report

**Date:** August 17, 2026
**Status:** Complete for the bounded local queue milestone.

## Completed

- Added the first Go Control Plane module and accepted ADR-0005.
- Added a loopback-only HTTP process with explicit header and connection
  timeouts.
- Added a minimal `Run Test` action to the existing result surface.
- Limited the browser request to the approved process-marker scenario and the
  registered lab environment and host.
- Added strict JSON, content-type, body-size, origin, CSRF, and unknown-field
  validation.
- Added idempotency-key fingerprinting and synchronous double-click protection.
- Added a bounded concurrency-safe queue and one-time host-bound lease
  interface.
- Added short-lived Ed25519-signed Test Jobs with empty parameters.
- Added expiry pruning and bounded idempotency retention.
- Added versioned JSON contracts for the create request and queue receipt.

## Acceptance result

| Acceptance item | Result |
|---|---|
| Browser can queue the fixed safe scenario | PASS |
| Browser cannot submit command text or arguments | PASS |
| Unknown JSON fields fail closed | PASS |
| Foreign origin or invalid CSRF fails closed | PASS |
| Concurrent duplicate requests create one job | PASS |
| Changed request with reused key conflicts | PASS |
| Job is signed, host-bound, and short-lived | PASS |
| Wrong host, second lease, and expired lease fail | PASS |
| Queue and idempotency storage are bounded | PASS |
| Control Plane remains loopback-only | PASS |

## Validation

- Full Control Plane Go test suite
- Go static analysis
- Dashboard request-boundary validator
- Repository whitespace validation
- Local HTTP session, create, idempotent replay, and static-interface smoke test

## Deliberately deferred

The queue acceptance result ends at `awaiting Runner lease`. Day 28 will add the
outbound Runner-facing lease/acknowledgement path and lifecycle state without
opening an inbound port on Windows. The static representative result remains in
place until live stage updates exist.

Customer outreach remains deferred under ADR-0002 because the current action
does not yet complete a visible end-to-end run.
