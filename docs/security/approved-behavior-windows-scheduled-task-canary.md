# Approved Behavior: Windows Scheduled Task Canary

**Decision:** Approved for isolated-lab implementation and verification
**Date:** August 13, 2026
**Owner:** Founding team
**Scenario:** `windows-scheduled-task-canary` version `0.1.0`

## Approved behavior

The built-in handler creates exactly one root-level Scheduled Task named
`ProofLayer_` plus the 128-bit canonical correlation identifier. The task uses
the fixed `ONLOGON` trigger, `LIMITED` run level, and a fixed harmless action
that exits immediately. The handler immediately deletes that exact task and
independently verifies that its Task Scheduler artifact file is absent.

## Safety boundary

- No caller-defined task name, folder, trigger, principal, run level, action,
  command, argument, payload, or schedule
- Task name derived only from a validated canonical correlation ID
- Fixed logon trigger; no immediate, recurring, remote, or event-based trigger
- Fixed limited run level; no SYSTEM principal or highest-privilege request
- Fixed harmless action: `C:\Windows\System32\cmd.exe /d /c exit 0`
- No network, customer data, credential, or persistence payload access
- Cleanup runs after every execution outcome, including cancellation and error
- Cleanup receives its own bounded context and does not inherit cancellation
- Artifact absence is verified separately through the fixed task artifact path
- A remaining task produces terminal `artifact_remaining`, never PASS

## Expected telemetry

- Provider: `Microsoft-Windows-TaskScheduler/Operational`
- Event IDs: `106` (task registered) and `141` (task deleted)
- Required fields: `host.name`, `task.name`, `user.name`
- Correlation: exact canonical `PL-` ID encoded in the task name

Telemetry availability depends on the Task Scheduler Operational channel and
is not part of the Day 25 artifact-cleanup acceptance proof.

## Approval limits

Approval applies only to the isolated ProofLayer lab and the fixed built-in
handler. It does not approve arbitrary task creation, remote scheduling,
production execution, recurring schedules, configurable actions, privilege
elevation, or persistence that survives the cleanup window.

## Revocation conditions

Approval is revoked if any task property becomes caller-controlled, the task
requests elevated privileges, cleanup can be skipped, absence is inferred
without a separate check, or task definitions and command output enter logs or
reports.
