# Phase 2 Evidence-Ranked Backlog

**Ranked:** August 5, 2026  
**Basis:** Day 6-14 technical evidence and frozen MVP Requirements v0.1

| Rank | Priority | Work item | Evidence unlocked |
|---:|---|---|---|
| 1 | P0 | Go Runner skeleton, identity model, and fixed allowlist | Safe executable product boundary |
| 2 | P0 | Canonical correlation ID and local audit event | Traceable run identity |
| 3 | P0 | Runtime/resource limits and cancellation tests | Bounded execution proof |
| 4 | P0 | Built-in Windows process-marker handler | Execution-stage evidence |
| 5 | P0 | Local Sysmon Observer | Endpoint-telemetry stage evidence |
| 6 | P0 | Read-only Splunk exact-ID connector | SIEM-ingestion stage evidence |
| 7 | P0 | Required-field validator | Parser/field stage evidence |
| 8 | P0 | Stage state machine and failure propagation | Product-level PASS/FAIL result |
| 9 | P0 | Cleanup and artifact-absence verifier | COMPLETE — Day 24 Registry and Day 25 Scheduled Task live proofs |
| 10 | P1 | Minimal run API and dashboard pipeline view | IN PROGRESS — Day 27 secure queue action complete; Runner lease and live stages remain |
| 11 | P1 | Redacted JSON/HTML export | Pilot evidence package |
| 12 | P1 | Clean-lab installer and demo runbook | Reproducible external demo |

## Deferred work

Detection-rule and alert connector abstractions follow the core five-stage MVP
proof. Additional scenarios, SIEMs, AI features, and customer-specific actions
remain outside this phase.

Customer tasks stay deferred under ADR-0002 until the visible MVP and cleanup
gate is complete.
