# Day 19 Completion Report

**Date:** August 5, 2026  
**Status:** Complete with live local Splunk validation.

## Completed

- Implemented the Go Splunk connector with HTTPS-only configuration, TLS 1.2
  minimum, a 10-second deadline, a 64 KiB response cap, and no URL credentials.
- Fixed runtime identity and scope to `prooflayer_observer` and
  `prooflayer_test`.
- Added an authenticated connection/permission check that returns only an
  aggregate count and no event body.
- Loaded the local observer password from one environment variable without
  logging, serializing, or returning it.
- Restricted the self-signed certificate exception to the explicit loopback
  lab flag and endpoint; trusted TLS remains the default.
- Added deterministic tests for correct request scope, invalid authorization,
  unavailable transport, unsafe configuration, and missing credentials.
- Built the connector check utility for Windows ARM64.

## Live validation

| Case | Result |
|---|---|
| Valid `prooflayer_observer` credential | PASS |
| Allowed index | `prooflayer_test` |
| Invalid observer credential | `authentication failed`, non-zero exit |
| Stopped Splunk endpoint | `endpoint unavailable`, non-zero exit |
| Admin credential supplied to connector | Never |
| HEC token supplied to connector | Never |

## Secret boundary

The founding lab stores the generated observer password only in ignored
`deployments/splunk/.env` and injects it into the check process environment.
Customer deployment must replace this local exception with an operating-system
secret store or dedicated secret manager. The connector API accepts the secret
in memory and has no method that prints or serializes its configuration.

## Minimum role

The existing `prooflayer_observer` role retains only the `search` capability,
allowed/default index `prooflayer_test`, and no inherited `admin`, `power`, or
`user` role. Its negative `_internal` acceptance test remains part of the lab
gate.

## Next gate

Add exact-correlation search, a bounded configurable search window, polling,
and late-event tests to the connector without returning raw endpoint values.
