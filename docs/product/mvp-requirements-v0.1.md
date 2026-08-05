# ProofLayer MVP Requirements v0.1

**Status:** Frozen  
**Freeze date:** August 5, 2026  
**Change control:** A requirement change requires an ADR.

## Product outcome

Prove, in a visible stage-level result, whether one approved Windows signal was
executed, generated endpoint telemetry, reached Splunk with required fields,
and left no artifact.

## Functional requirements

| ID | Requirement | MVP acceptance |
|---|---|---|
| FR-01 | Runner identity | Runner registers with an opaque stable ID and authenticated environment binding. |
| FR-02 | Fixed scenario catalog | Only signed, versioned built-in scenarios are resolvable; arbitrary commands are rejected. |
| FR-03 | Safe execution | The approved process-marker handler runs with a deadline and canonical correlation ID. |
| FR-04 | Endpoint observation | Exact-ID Sysmon Event ID 1 is found with bounded local observation. |
| FR-05 | Splunk observation | A read-only Observer finds exactly one event in `prooflayer_test`. |
| FR-06 | Field validation | Four required field names are reported as present or missing without returning their values. |
| FR-07 | Stage state machine | Execution, telemetry, ingestion, field validation, detection, alert, and cleanup follow the frozen contract. |
| FR-08 | Cleanup | Cleanup runs after every terminal path and proves artifact absence. |
| FR-09 | Run history | The operator can view the stage result, timestamps, latencies, stable error, and evidence references. |
| FR-10 | Report | One redacted JSON and HTML report can be exported for a run. |

## Security requirements

| ID | Requirement |
|---|---|
| SR-01 | No arbitrary shell or scenario-supplied executable/arguments. |
| SR-02 | Signed jobs, expiry, nonce protection, host binding, and fail-closed validation. |
| SR-03 | Outbound authenticated Runner communication only; no inbound Runner listener. |
| SR-04 | Least-privilege Observer and ingestion identities. |
| SR-05 | Kill switch, maintenance window, runtime/resource limits, and cancellation. |
| SR-06 | Minimum metadata only; no raw events, command-line values, usernames, or secrets centrally stored. |
| SR-07 | Local tamper-evident audit trail and redacted bounded errors. |

## Quality requirements

- Three consecutive lab runs pass the same observation contract.
- The same correlation ID cannot create more than one successful run result.
- A missing field produces `field_validation=failed`; downstream stages become
  `not_tested` according to the stage contract.
- A clean-lab installation and demo can be reproduced from documented steps.
- Unit, contract, security-negative, and connector smoke tests run in CI.

## Explicitly out of scope

- Exploitation, credential access, ransomware, or destructive simulation
- Arbitrary customer-authored commands
- Cloud, Linux, Kubernetes, OT, EDR, SOAR, or multi-SIEM support
- Automated remediation or autonomous AI execution
- Attack-path or vulnerability analysis
- Production customer deployment before the pilot security gate

## MVP demonstration gate

The MVP is demonstrable only when one operator action produces a visible result
for execution, endpoint telemetry, Splunk ingestion, field validation, and
cleanup. Detection and alert may be shown as `not_tested` until their bounded
connector contracts are implemented.
