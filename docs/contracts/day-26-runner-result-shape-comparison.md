# Day 26 Runner Result Shape Comparison

**Compared:** August 17, 2026
**Contract:** `runner-execution-result.schema.json` version `1.0`

## Result

All three approved scenarios emit the same Runner execution-result shape. A
consumer does not branch on scenario type to decode the envelope.

| Field | Process marker | Registry canary | Scheduled Task canary |
|---|---|---|---|
| `schema_version` | `1.0` | `1.0` | `1.0` |
| `status` | Required | Required | Required |
| `correlation_id` | Required | Required | Required |
| `scenario_id` | Required | Required | Required |
| `scenario_version` | Required | Required | Required |
| `started_at` | Required | Required | Required |
| `completed_at` | Required | Required | Required |
| `latency_ms` | Required | Required | Required |
| `cleanup_status` | Required | Required | Required |
| `error_code` | Failure only | Failure only | Failure only |

Scenario behavior, telemetry expectations, and cleanup strategies remain in the
signed scenario definition. They do not add arbitrary fields to the result.

## Compatibility rule

- Additive optional fields require a documented minor contract revision.
- Removing, renaming, retyping, or changing the meaning of a field requires a
  new major schema version.
- A successful result must omit `error_code`.
- A failed result must use one of the stable allowlisted error codes.
- Raw command output, task definitions, Registry data, credentials, and event
  bodies are prohibited from this result envelope.

## Product boundary

The Runner result reports bounded execution and cleanup. The richer
`test-run.schema.json` contract owns endpoint telemetry, SIEM ingestion, field
validation, detection, alert, root-cause, and product-facing pipeline stages.
The Day 26 wireframe represents that richer product contract using synthetic
data.
