# Day 27 Run Test Request Boundary

**Date:** August 17, 2026
**Scope:** Local Control Plane PoC and one fixed browser action

## Security objective

The browser can request one approved scenario for one registered lab host. It
cannot submit command text, arguments, executable content, parameters, a new
host, or a new scenario. Accepting the browser request creates a short-lived,
host-bound, Ed25519-signed job; it does not execute the scenario directly.

## Request flow

```text
Local browser
  → same-origin CSRF session
  → strict POST /v1/test-jobs
  → fixed environment and host validation
  → scenario/version allowlist
  → idempotency check
  → two-minute signed job
  → bounded in-memory queue
  → one-time host-bound lease interface
```

The Runner still initiates transport. Day 27 does not expose an inbound Runner
listener and does not add a browser-to-Runner path.

## Implemented controls

| Risk | Control | Negative or concurrency proof |
|---|---|---|
| Cross-site request | Exact loopback `Origin`, `Sec-Fetch-Site`, SameSite HttpOnly cookie, constant-time CSRF token check | Missing/foreign origin and missing/wrong token rejected |
| Parser ambiguity | JSON-only media type, 4096-byte maximum, unknown fields rejected, exactly one JSON value | Unknown `command`, wrong media type, and oversized body rejected |
| Arbitrary execution | Server-side scenario/version allowlist; request has no command, arguments, payload, or parameters field | Unknown scenario and version rejected |
| Wrong target | Environment and host are bound in the queue configuration | Foreign environment or host rejected |
| Double click or retry | Synchronous UI disable plus server-side idempotency fingerprint | Twelve concurrent identical requests create one job |
| Replay with changed body | One key is bound to one normalized request fingerprint | Changed request returns conflict |
| Queue exhaustion | Fixed queue capacity and fixed idempotency-record ceiling | Capacity overflow returns unavailable |
| Stale authorization | Two-minute expiry and pruning before lease | Expired job cannot be leased |
| Job tampering | Ed25519 signature over the full canonical job authorization | Signature verification test passes |
| Cross-host lease | Lease matches both environment and host and removes the job once | Wrong identity and second lease fail |

## Local PoC limitations

- The server binds only to `127.0.0.1:8787` and uses HTTP. The session cookie is
  intentionally not marked `Secure` because local HTTP would reject it.
- The signing key and queue are ephemeral. Restarting the process invalidates
  queued work.
- There is no operator login or RBAC. Loopback binding is not a production
  authentication mechanism.
- The single-file interface currently requires an inline-script CSP exception.
- The Runner lease transport, acknowledgement, execution, stage updates,
  persistence, audit chain, and cancellation are subsequent work.

This process must not be exposed to a LAN, public network, customer environment,
or production workload.
