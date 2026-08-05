# Runner Execution Limits v0.1

**Status:** Frozen for `windows-process-marker@0.1.0`  
**Date:** August 5, 2026

## Approved limits

| Control | Limit |
|---|---:|
| Execution deadline | 30 seconds |
| Cleanup deadline | 10 seconds |
| Captured output | 4096 bytes maximum |
| Child processes | Exactly one |
| Network access | None |
| Scenario parameters | Empty object |

The policy validator rejects any attempt to relax a limit. The executor must
derive its cancellation context from this policy and may enforce a stricter
limit. The Day 17 Windows handler must discard child output and apply an
operating-system process boundary before it can be approved for execution.

## Correlation ID

The Runner generates 128 random bits with the operating-system cryptographic
random source and encodes them as `PL-` plus 32 uppercase hexadecimal
characters. Random-source failure stops the run; there is no fallback generator.

## Local audit event

Audit records are append-only JSON Lines with restricted local permissions.
They contain stable IDs, timestamps, outcome, and bounded error codes only. They
exclude commands, parameters, event bodies, usernames, secrets, and arbitrary
process output.
