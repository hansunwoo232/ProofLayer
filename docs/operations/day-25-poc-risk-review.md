# Day 25 PoC Risk Review

**Reviewed:** August 13, 2026
**Scope:** Fixed Windows Scheduled Task canary in the isolated QEMU lab
**Decision:** Proceed to isolated live proof with the controls below.

## Open risks and dispositions

| Risk | Current control | Residual disposition |
|---|---|---|
| Task remains after an interrupted run | Cleanup uses a separate 10-second context and verifies the artifact path | Accept for isolated lab; any residue is terminal FAIL and must be removed before checkpointing |
| Task fires before deletion | Trigger is fixed to `ONLOGON`; action only exits successfully | Accept; do not log off or restart while the scenario is running |
| Privilege escalation through the task | Run level is fixed to `LIMITED`; no SYSTEM or alternate principal | Accept |
| Arbitrary command or task injection | All properties are compiled constants; only the validated correlation ID affects the name | Accept |
| Security tooling reports the canary as suspicious | Dedicated `ProofLayer_` prefix and correlation-bound audit events | Accept in isolated lab; do not globally suppress alerts |
| Task Scheduler service or permissions prevent cleanup | Fail-closed result plus independent COM and filesystem checks | Accept; no checkpoint is allowed after a failed cleanup |
| Operational event channel is disabled | Day 25 acceptance requires artifact cleanup, not telemetry ingestion | Track for the later telemetry-validation pass |
| Lab code is mistaken for production readiness | Lab harness accepts no arguments and production CLI exposes no execute command | Accept; production authorization and signing remain required work |

## Exit criteria

- Runner status is PASS.
- Runner cleanup status is PASS.
- Independent Task Scheduler COM lookup reports absent.
- Independent task artifact file check reports absent.
- Windows shuts down normally before the Day 25 checkpoint.

Any failed exit criterion blocks the checkpoint and requires investigation.
