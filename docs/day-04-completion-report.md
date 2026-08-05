# Day 4 Completion Report

**Date:** August 2, 2026  
**Status:** Complete. Customer discovery remains deferred under ADR-0002.

## Completed

### Sysmon process telemetry

- Installed Sysmon v15.21 Arm64 from read-only offline media.
- Applied the minimal ProofLayer process-marker configuration.
- Standardized the dedicated Windows lab on UTC.
- Generated correlation ID `PL-TEST-BOOTSTRAP-7E31F075A720`.
- Verified the marker in Sysmon Event ID 1, Record ID 137.
- Created the `sysmon-baseline` checkpoint after successful validation.

### Scenario safety contract

- Added explicit built-in handler and network-scope allowlists.
- Made cleanup mandatory with bounded timeout and retry counts.
- Required cleanup to verify artifact absence.
- Bound each action to its only permitted handler, network policy, and cleanup
  strategy through JSON Schema conditions.
- Added the first YAML scenario and negative safety-mutation tests.

### Runner–Control Plane protocol

- Defined outbound-only HTTPS/mTLS transport and host identity.
- Defined registration, heartbeat, job lease, acknowledgement, stage update,
  completion, cancellation, and kill-switch exchanges.
- Defined local job authorization order, replay protection, state transitions,
  idempotency, retry behavior, stable error codes, and the evidence boundary.

## Validation

```text
PASS scenario example
PASS test job example
PASS parser-failure run example
PASS cross-contract security invariants
PASS unsafe scenario mutations rejected
```

## Deferred by decision

The two planned customer discovery interviews were not performed. ADR-0002
defers all external outreach until the repeatable MVP is ready.

## Day 5 inputs

- Known-good Windows and Sysmon baseline
- Safe process-marker YAML contract
- Runner–Control Plane protocol v0.1
- Correlation ID and lifecycle rules ready for Splunk lab integration
