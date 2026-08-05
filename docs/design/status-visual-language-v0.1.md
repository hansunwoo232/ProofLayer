# Pipeline Status Visual Language v0.1

**Status:** Frozen for the MVP result view  
**Date:** August 5, 2026

## Status tokens

| Machine status | Visible label | Tone | Symbol | Meaning |
|---|---|---|---|---|
| `passed` | PASS | Positive | Check | Required evidence satisfied the stage contract. |
| `failed` | FAIL | Critical | X | The stage ran and did not satisfy its contract. |
| `not_tested` | NOT TESTED | Neutral | Minus | The stage did not run; usually an upstream failure prevented it. |
| `degraded` | DEGRADED | Warning | Warning | Evidence passed but an approved SLO was exceeded. |
| `running` | RUNNING | Informative | Progress | The stage is actively executing or observing. |
| `pending` | PENDING | Neutral | Clock | The stage has not started. |

Color is never the only carrier of state. Every state uses a text label, symbol,
and accessible description. `NOT TESTED` must never be rendered as FAIL or PASS.

## Pipeline presentation

1. Always render all seven stages in the frozen order.
2. Keep cleanup visible as the final safety stage even after an upstream FAIL.
3. Emphasize the first failed required stage as the root-cause boundary.
4. Render prevented downstream stages as NOT TESTED.
5. Show a bounded error code and plain-language summary; never show raw event
   bodies, commands, usernames, or secret-bearing transport errors.
6. Do not animate terminal states. RUNNING may use a reduced-motion-safe
   progress treatment.

## Overall result

- FAIL if any required stage or cleanup fails.
- DEGRADED if no required stage fails and at least one stage is degraded.
- PASS only after cleanup passes and every required stage has passed or met its
  explicitly approved degraded condition.
- RUNNING while required work or cleanup remains active.
