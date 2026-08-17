# Runner Execution Limits v0.1

**Status:** Frozen for `windows-process-marker@0.1.0`,
`windows-registry-run-key-canary@0.1.0`, and
`windows-scheduled-task-canary@0.1.0`
**Updated:** August 13, 2026

## Approved limits

| Control | Limit |
|---|---:|
| Execution deadline | 30 seconds |
| Cleanup deadline | 10 seconds |
| Captured output | 4096 bytes maximum |
| Child processes | Exactly one |
| Network access | None |
| Scenario parameters | Empty object |

The policy validator rejects any attempt to relax a limit. The executor derives
execution and cleanup contexts from this policy and may enforce stricter
limits. Cleanup uses a separate bounded context that does not inherit caller
cancellation, so cancellation cannot skip artifact removal.

The process-marker handler discards child output and applies an operating-system
process boundary. The Registry canary uses direct Windows Registry APIs, accepts
no caller-defined path or value, and verifies absence after deletion.
The Scheduled Task canary invokes only the fixed Windows task utility with
compiled arguments, discards all child output, and verifies the fixed task
artifact path after deletion. Creation and cleanup each run at most one child
process in their separate bounded phases.

## Correlation ID

The Runner generates 128 random bits with the operating-system cryptographic
random source and encodes them as `PL-` plus 32 uppercase hexadecimal
characters. Random-source failure stops the run; there is no fallback generator.

## Local audit event

Audit records are append-only JSON Lines with restricted local permissions.
They contain stable IDs, timestamps, outcome, and bounded error codes only. They
exclude commands, parameters, event bodies, usernames, secrets, and arbitrary
process output.
