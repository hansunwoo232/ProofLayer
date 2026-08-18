# Runner–Control Plane Protocol v0.1

**Status:** Draft for the technical PoC  
**Protocol version:** `1.0`

## Purpose

This protocol lets a registered Windows Runner receive a short-lived,
cryptographically authorized Test Job and return minimum execution evidence.
The Runner never exposes an inbound listener and never accepts command text,
PowerShell text, executable content, or an unknown scenario handler.

## Transport and identity

- The Runner initiates every connection to the Control Plane.
- Production transport is HTTPS with mutual TLS.
- A client certificate identifies one organization, environment, and host.
- The local PoC may use one identity-bound bearer credential only while both
  endpoints are on loopback. Plain HTTP to a non-loopback address is rejected.
- JSON payloads use `Content-Type: application/json` and the versioned schemas
  in `schemas/v1/`.
- All timestamps are RFC 3339 UTC.
- Requests use a unique `message_id` and an idempotency key.
- The Runner keeps no inbound firewall exception.

## Message envelope

```json
{
  "protocol_version": "1.0",
  "message_id": "7c6895f5-72f6-4ce8-a1a6-650a5554491e",
  "sent_at": "2026-08-01T21:57:31Z",
  "idempotency_key": "job-56a9c2f7-stage-execution-1",
  "payload": {}
}
```

Unknown envelope or payload fields are rejected at the trust boundary.

## Registration

1. An operator creates a single-use bootstrap token scoped to one environment.
2. The Runner generates its private key locally and submits a certificate
   signing request to `POST /v1/runners/register`.
3. The Control Plane consumes the bootstrap token and returns `runner_id`,
   `host_id`, the client certificate chain, policy version, and polling limits.
4. The private key never leaves the host.

Bootstrap tokens expire within 15 minutes and cannot be reused.

## Core exchanges

| Method and path | Direction | Purpose |
|---|---|---|
| `POST /v1/runners/register` | Runner → Control Plane | One-time host registration |
| `POST /v1/runners/{runner_id}/heartbeat` | Runner → Control Plane | Health, version, and policy state |
| `POST /v1/runners/{runner_id}/jobs:lease` | Runner → Control Plane | Long-poll for one eligible job |
| `POST /v1/runners/{runner_id}/jobs/{job_id}:ack` | Runner → Control Plane | Accept or reject the leased job |
| `PUT /v1/runners/{runner_id}/jobs/{job_id}/stages/{stage}` | Runner → Control Plane | Idempotent stage result update |
| `POST /v1/runners/{runner_id}/jobs/{job_id}:complete` | Runner → Control Plane | Final result and cleanup status |
| `GET /v1/runners/{runner_id}/control` | Runner → Control Plane | Poll kill-switch and cancellation state |
| `GET /v1/test-jobs/{job_id}` | Operator browser → Control Plane | Read one session-bound live status snapshot |

The PoC uses bounded long polling. A streaming channel is explicitly deferred.

## Day 28 lifecycle implementation

The Control Plane now models `queued → leased → acknowledged → running →
completed/failed` with terminal `rejected` and `expired` branches. Stage updates
are ordered, identity-bound, latency-bounded, and limited to stable detail
codes. Upstream failure prohibits downstream PASS results but does not skip
cleanup. The browser receives only a read-only snapshot and cannot mutate the
Runner lifecycle.

## Day 29 transport implementation

The local Control Plane exposes lease, acknowledgement, ordered stage-update,
and completion routes only when an explicit Runner binding is configured. One
bearer credential is bound server-side to a canonical Runner, environment, and
host identity and is compared in constant time before a lease can change queue
state. Missing, wrong, and cross-Runner credentials fail before job lookup.

The outbound Runner client accepts plain HTTP only for an IP loopback endpoint;
all other endpoints require HTTPS. It uses bounded requests, strict JSON,
response-size limits, and a fixed request model. A leased job must pass local
Ed25519 verification, identity and expiry checks, built-in scenario resolution,
empty-parameter enforcement, and in-memory nonce replay protection before it
can become an execution request.

This is a PoC transport boundary, not the production identity design. mTLS,
certificate registration and rotation, durable nonce state, bounded retry, and
Windows worker orchestration remain required before customer deployment.

## Job authorization order

The Runner performs every check locally before any scenario handler starts:

1. Parse the envelope and validate `test-job.schema.json`.
2. Verify the Ed25519 signature over the canonical job payload.
3. Confirm `host_id` and `environment_id` match the local identity.
4. Reject expired jobs or unacceptable clock drift.
5. Reject a previously consumed `job_id` or nonce.
6. Resolve `scenario_id` and version from the local signed catalog.
7. Confirm the action, built-in handler, network scope, and cleanup strategy
   match the fixed Runner allowlist.
8. Validate parameters against the scenario parameter schema.
9. Confirm the maintenance window and local kill switch allow execution.
10. Acquire the single-execution lock and acknowledge the job.

Any failure is fail-closed. There is no fallback to a shell.

## Runner state machine

```text
idle → leased → validated → acknowledged → running → cleaning_up → completed
                 ↘ rejected
running → cancelling → cleaning_up → cancelled
```

Cleanup runs after success, execution failure, timeout, or cancellation. A
cleanup failure makes the overall Test Run fail.

## Retry and idempotency

- The Control Plane may redeliver an unacknowledged lease.
- A consumed nonce can never start a second execution.
- Stage updates are idempotent by `run_id + stage + attempt`.
- The Runner retries only transport failures and `429`/`5xx` responses.
- Exponential backoff is bounded and honors `Retry-After`.
- Results are buffered locally with a size and retention limit when the Control
  Plane is temporarily unavailable.

## Error model

```json
{
  "code": "JOB_SIGNATURE_INVALID",
  "message": "The Test Job signature could not be verified.",
  "retryable": false
}
```

Stable error codes include `SCHEMA_INVALID`, `JOB_EXPIRED`, `HOST_MISMATCH`,
`NONCE_REPLAYED`, `SCENARIO_NOT_ALLOWED`, `PARAMETER_REJECTED`,
`MAINTENANCE_WINDOW_CLOSED`, `KILL_SWITCH_ACTIVE`, and `CLEANUP_FAILED`.

## Evidence boundary

The default protocol sends identifiers, timestamps, stage status, latency,
event counts, field names, Event ID, Record ID, and hashes. It does not send raw
event bodies, credentials, arbitrary command lines, customer file contents, or
security-product secrets.

## PoC acceptance criteria

- A valid job can move from lease to completed exactly once.
- An expired, replayed, mismatched, or incorrectly signed job is rejected.
- Unknown actions and mismatched built-in handlers are rejected locally.
- Cancellation still executes and verifies cleanup.
- Repeated result delivery does not create duplicate runs or stages.
- No inbound Runner port is opened.

## Deferred decisions

- Certificate authority and rotation implementation
- Maximum offline result buffer and retention period
- Exact long-poll duration and heartbeat interval
- Accepted clock-drift threshold
- Streaming transport after the PoC
