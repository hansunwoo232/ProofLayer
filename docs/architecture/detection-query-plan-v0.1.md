# Detection Query Plan v0.1

**Status:** Frozen for `windows-process-marker@0.1.0`  
**Date:** August 5, 2026

## Detection identity

- Detection ID: `prooflayer.windows_process_marker`
- Required upstream stages: execution, endpoint telemetry, SIEM ingestion, and
  field validation must pass
- Result rows: zero or one accepted; two is ambiguous and fails closed

## Supported plans

### Built-in inline plan

The connector constructs one fixed SPL query against `prooflayer_test` and
`prooflayer:windows-lab`. It validates the exact correlation ID and Event ID 1,
then emits only `correlation_id` and the constant `detection_id`.

### Fixed Saved Search plan

The connector can invoke only the Saved Search named
`ProofLayer Windows Process Marker`. It passes three validated values:

- Canonical correlation ID
- Numeric earliest UTC epoch
- Numeric latest UTC epoch

The Saved Search name and detection ID are compiled into the Runner. No API
accepts a Saved Search name, SPL fragment, field expression, command, or macro
from a scenario or operator.

## Result semantics

| Matches | Result |
|---:|---|
| 0 | `failed`, `detected=false` |
| 1 exact row | `passed`, `detected=true` |
| 2 rows | `ambiguous`; no stage PASS |

Authentication, connection, malformed JSON, result-ID mismatch, invalid plan,
or an unbounded window is an evaluation error rather than a false detection
failure.

## Data boundary

The returned evidence contains status, detected boolean, correlation ID,
detection ID, plan mode, fixed rule reference, and match count. It cannot
contain `_raw`, command lines, usernames, hostnames, or event bodies.
