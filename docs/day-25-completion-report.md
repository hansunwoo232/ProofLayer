# Day 25 Completion Report

**Date:** August 13, 2026
**Status:** Complete within the approved isolated-lab technical workstream.

## Completed

- Added `windows-scheduled-task-canary@0.1.0` to the fixed built-in catalog.
- Restricted the task name to `ProofLayer_` plus the validated canonical
  correlation identifier.
- Fixed the trigger to `ONLOGON`, run level to `LIMITED`, and action to a
  harmless command that exits immediately.
- Accepted no caller-defined task, trigger, principal, action, command,
  argument, payload, schedule, or path.
- Reused mandatory cleanup with a separate bounded cleanup context.
- Deleted the exact correlation-bound task and verified its artifact file was
  absent before returning PASS.
- Added the scenario contract, security approval, PoC risk review, lab harness,
  PowerShell wrapper, and read-only ISO build path.

## Live Windows proof

| Evidence | Result |
|---|---|
| Scenario | `windows-scheduled-task-canary@0.1.0` |
| Correlation ID | `PL-0949098C4348056494B032162D504E34` |
| Execution and cleanup latency | 117 ms |
| Runner status | PASS |
| Runner cleanup status | PASS |
| Independent Task Scheduler COM lookup | PASS — task absent |
| Independent task artifact file check | PASS — file absent |
| Trigger | Fixed `ONLOGON` |
| Run level | Fixed `LIMITED` |

The local screenshot is stored only in the ignored evidence directory as
`day-25-scheduled-task-cleanup-pass.png`. Its SHA-256 is
`58e14fa2936737b678b507cfaa917c1edf11ec42fdeaaafe5062629ffc6d54ab`.

After a clean Windows shutdown, the local lab disk, UEFI, and TPM state were
captured in the ignored checkpoint named
`day-25-scheduled-task-cleanup-proof`.

## Regression coverage

- Correlation-bound task-name construction
- Invalid correlation ID rejection
- Exact create and delete argument allowlists
- Empty scenario parameter enforcement
- Mandatory cleanup and artifact-absence state handling
- Catalog fail-closed lookup
- Windows ARM64 cross-build
- Scenario contract safety validation

## PoC risk review

The open risks and their dispositions are recorded in
`docs/operations/day-25-poc-risk-review.md`. No risk permits production use,
arbitrary task creation, elevated principals, recurring execution, skipped
cleanup, or customer environment access.

## Deferred under ADR-0002

No customer outreach was sent. The visible product-facing stage view remains
the open demonstration-gate item.

## Next technical target

Day 26 versions the test-result JSON output, compares all three scenario result
shapes, and produces the first simple result-screen wireframe.
