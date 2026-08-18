# Day 29 Completion Report

**Date:** August 18, 2026  
**Status:** Complete for the authenticated outbound transport boundary.

## Completed

- Added Runner-only lease, acknowledgement, ordered stage-update, and
  completion HTTP routes.
- Kept every Runner mutation route disabled unless an explicit Runner binding
  and high-entropy bearer credential are configured.
- Bound the credential server-side to one canonical Runner, environment, and
  host identity.
- Authenticated requests before leasing or looking up a Test Job.
- Added strict, size-limited, versioned JSON request contracts for every Runner
  lifecycle mutation.
- Added an outbound Runner client with bounded HTTP behavior and stable remote
  errors.
- Restricted plain HTTP Control Plane URLs to IP loopback; non-loopback targets
  require HTTPS.
- Added local Ed25519 signature, identity, expiry, built-in allowlist, empty
  parameter, and nonce-replay checks for leased jobs.
- Converted a verified Test Job into the existing fixed execution-request
  shape without adding command text or general-purpose payload support.
- Recorded ADR-0006 and updated the protocol, component documentation, JSON
  contracts, and evidence-ranked backlog.

## Acceptance result

| Acceptance item | Result |
|---|---|
| Missing or wrong Runner credential cannot consume a lease | PASS |
| A credential is bound to one Runner, environment, and host | PASS |
| One authenticated request leases one signed job | PASS |
| Strict ack, stage, and complete requests drive the ordered lifecycle | PASS |
| Unknown JSON fields and out-of-order transitions fail closed | PASS |
| Tampered job signature is rejected locally | PASS |
| Wrong-host and expired jobs are rejected locally | PASS |
| A consumed nonce cannot be accepted twice | PASS |
| Arbitrary Runner detail text is rejected | PASS |
| Plain HTTP to a non-loopback Control Plane is rejected | PASS |
| No inbound Windows Runner listener was added | PASS |

## Validation

- Full Control Plane Go unit and lifecycle test suite
- Full Runner Go unit test suite
- Runner transport authorization, malformed-body, and ordered-lifecycle tests
- Runner client signature, identity, expiry, replay, URL-policy, and remote
  error tests
- Go race detector and static analysis
- Versioned JSON contract and repository whitespace validation

## Deliberately deferred

The production Runner CLI does not yet start the worker loop. Day 30 connects a
verified lease to the existing fixed executor, Sysmon observer, Splunk search,
field validator, and lifecycle client so one browser action can produce a real
terminal result.

The Windows VM boundary still requires HTTPS/mTLS or an equivalently
authenticated lab design; the same-host HTTP bearer mode must not be exposed to
the VM network. Durable registration, token or certificate rotation, durable
nonce state, bounded retry, cancellation, and offline buffering remain later
milestones.

Customer demos and outreach remain deferred under ADR-0002 until the Day 30
end-to-end gate passes.
