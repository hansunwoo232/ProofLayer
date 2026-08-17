# Day 24 Completion Report

**Date:** August 11, 2026
**Status:** Complete within the approved technical workstream.

## Completed

- Added `windows-registry-run-key-canary@0.1.0` to the fixed built-in catalog.
- Implemented the canary with direct Windows Registry APIs and no generic
  registry, shell, command, path, payload, or parameter primitive.
- Restricted mutation to one correlation-bound value under
  `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`.
- Used fixed harmless value data and no network access.
- Refactored every built-in scenario around an explicit
  `execute → cleanup → verify absent` contract.
- Made cleanup run after successful execution, failed execution, timeout, and
  caller cancellation.
- Gave cleanup its own bounded context so parent cancellation cannot skip it.
- Added stable `cleanup_failed` and `artifact_remaining` terminal error codes.
- Added the scenario YAML, JSON contract example, lab harness, PowerShell
  wrapper, and read-only ISO build path.

## Live Windows proof

| Evidence | Result |
|---|---|
| Scenario | `windows-registry-run-key-canary@0.1.0` |
| Correlation ID | `PL-F2823DE375B9E68556B3682E6EAC2B71` |
| Execution and cleanup latency | 6 ms |
| Runner status | PASS |
| Runner cleanup status | PASS |
| Independent PowerShell artifact query | PASS — value absent |
| Registry boundary | Fixed HKCU Run path |

The local screenshot is stored only in the ignored evidence directory as
`day-24-registry-cleanup-pass.png`. Its SHA-256 is
`97670714e1f82bacdd45b023fd77f91b970f17ae7818c874ef89b7345ce55b4b`.

After a clean Windows shutdown, the local lab disk, UEFI, and TPM state were
captured in the ignored checkpoint named
`day-24-registry-cleanup-proof`.

## Regression coverage

- Approved scenario execution and cleanup PASS
- Cleanup after execution failure
- Cleanup operation failure
- Artifact still present after cleanup
- Parent cancellation does not cancel cleanup
- Cleanup deadline enforcement
- Invalid correlation ID rejection
- Defensive catalog lookup and fixed handler mapping
- Windows ARM64 cross-build
- Scenario contract safety validation

## Deferred under ADR-0002

No technical scope was sent to a pilot candidate. Customer outreach remains
deferred until the visible product-facing stage view completes the MVP demo
gate.

## Next technical target

Day 25 adds the fixed Scheduled Task create/delete canary with the same
mandatory cleanup and independent artifact-absence proof.
