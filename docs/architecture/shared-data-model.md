# Shared Data Model v1

ProofLayer uses versioned, language-neutral JSON contracts between the control
plane, Runner, Observer, and dashboard.

## Design rules

- All timestamps use RFC 3339 UTC.
- IDs are opaque and globally unique.
- `correlation_id` follows `PL-` plus 32 uppercase hexadecimal characters,
  carries 128 bits of randomness, is generated per run, and is never reused.
- Every payload includes `schema_version`.
- Unknown fields are rejected at security-sensitive boundaries.
- Raw customer event content is not part of the default result contract.
- Cleanup is a first-class stage, not an implicit side effect.
- A failed upstream stage makes dependent stages `not_tested`, not `failed`.

## Core models

### Scenario Definition

Describes an approved, versioned, safe behavior. It contains a typed action
identifier rather than an arbitrary command.

### Test Job

Authorizes one scenario execution on one registered host for a short time
window. The signature covers the host, scenario version, expiry, nonce, and
parameters.

### Stage Result

Records the status, timing, and minimum evidence for one pipeline stage:

```text
execution → endpoint_telemetry → siem_ingestion → field_validation
          → detection → alert → cleanup
```

### Test Run

Aggregates the signed job identity, host/scenario context, stage results, root
cause, and overall result.

## Overall result rules

1. `failed`: any required stage failed or cleanup failed.
2. `degraded`: no required stage failed, but at least one stage exceeded its SLO.
3. `passed`: every required stage passed and cleanup passed.
4. `running`: at least one required stage is pending/running.
5. `cancelled`: the operator or kill switch stopped the run.
6. `expired`: the job expired before execution.

## Root-cause categories

- `execution_error`
- `missing_endpoint_telemetry`
- `siem_ingestion_timeout`
- `missing_required_field`
- `detection_not_triggered`
- `alert_not_created`
- `cleanup_failed`
- `observer_error`
- `unknown`

## Contract files

- `schemas/v1/scenario.schema.json`
- `schemas/v1/test-job.schema.json`
- `schemas/v1/test-run.schema.json`
- `schemas/v1/examples/`

The generation, propagation, expiry, and redaction rules are defined in
`docs/architecture/correlation-id-lifecycle.md`.
