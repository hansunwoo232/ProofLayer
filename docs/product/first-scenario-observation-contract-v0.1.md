# First Scenario Observation Contract v0.1

**Status:** Frozen for the technical PoC  
**Date:** August 5, 2026  
**Scenario:** `windows-process-marker` version `0.1.0`

## Purpose

Define the exact evidence required to prove that the first safe scenario was
observed at the endpoint and ingested by Splunk. This contract does not claim
that a detection rule or alert fired.

## Exact-match requirements

| Requirement | Expected value |
|---|---|
| Provider | `Microsoft-Windows-Sysmon` |
| Event ID | `1` (Process Create) |
| Correlation | One canonical `PL-` identifier with 32 uppercase hex characters |
| Splunk index | `prooflayer_test` |
| Search result | Exactly one matching event for a run |
| Search window | A bounded 24-hour PoC window |
| Required fields | `host.name`, `process.name`, `process.command_line`, `user.name` |

The Observer returns the matched field names and counts. It does not return
command-line values, usernames, or raw event bodies.

## Stage decisions

- `endpoint_telemetry` passes only when the Runner's local bounded query finds
  one Sysmon Event ID 1 containing the exact correlation ID.
- `siem_ingestion` passes only when the read-only Splunk Observer finds exactly
  one matching event in `prooflayer_test`.
- `field_validation` passes only when all four required fields are present and
  non-empty inside Splunk.
- `detection` and `alert` remain `not_tested` in this contract.

## Failure conditions

- Zero or multiple exact-ID matches
- Provider or Event ID mismatch
- A missing required field
- Observer timeout, authorization failure, or result truncation
- A search outside the allowed index or time boundary
- Any evidence response containing raw command-line or username values

## Cleanup expectation

The behavior writes no persistent file, registry value, task, service, account,
or network configuration. Cleanup therefore verifies the absence of scenario
artifacts and records that verification as a separate terminal stage.

## Executable proof

`deployments/splunk/verify-windows-event-by-id.sh` implements the exact-ID
Splunk portion of this contract. It fails closed unless there is one result and
all required fields are present.
