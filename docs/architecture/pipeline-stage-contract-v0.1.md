# Pipeline Stage Contract v0.1

**Status:** Frozen for schema version `1.0`  
**Date:** August 5, 2026

| Order | Stage | Pass condition | Representative failure |
|---:|---|---|---|
| 1 | `execution` | The fixed built-in handler completes within policy limits. | `execution_error` |
| 2 | `endpoint_telemetry` | The expected provider and Event ID contain the exact correlation ID. | `missing_endpoint_telemetry` |
| 3 | `siem_ingestion` | Exactly one matching event reaches the configured index in the bounded window. | `siem_ingestion_timeout` |
| 4 | `field_validation` | Every scenario-required field exists and satisfies its typed presence rule. | `missing_required_field` |
| 5 | `detection` | The configured detection search returns the expected match. | `detection_not_triggered` |
| 6 | `alert` | The expected alert or notable-event reference is created when required. | `alert_not_created` |
| 7 | `cleanup` | Cleanup completes and artifact absence is independently verified. | `cleanup_failed` |

## Status vocabulary

- `pending`
- `running`
- `passed`
- `failed`
- `degraded`
- `not_tested`

## Propagation rules

1. Stage order cannot change within schema version `1.0`.
2. A failed required stage prevents dependent stages from running.
3. Prevented dependent stages are `not_tested`, never `failed`.
4. Cleanup remains required regardless of previous outcomes.
5. A cleanup failure makes the overall run fail.
6. Detection and alert may be configured as not required for an ingestion-only
   technical proof, but their positions remain reserved.
7. `passed` requires all required stages and cleanup to pass.
