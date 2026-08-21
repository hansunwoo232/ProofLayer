# Decision Log

| ID | Date | Decision | Status | Revisit |
|---|---|---|---|---|
| [ADR-0001](0001-runner-execution-boundary.md) | 2026-07-28 | The Runner will not execute arbitrary commands | Accepted | Day 16 and before the pilot |
| [ADR-0002](0002-defer-outreach-until-mvp.md) | 2026-07-29 | Defer customer outreach until the MVP is demonstrable | Accepted | First repeatable Windows-to-Splunk run |
| [ADR-0003](0003-splunk-least-privilege-observer.md) | 2026-08-02 | Separate Splunk ingestion and observation identities | Accepted | Before customer deployment |
| [ADR-0004](0004-single-egress-lab-network.md) | 2026-08-02 | Use a single-egress Windows-to-HEC lab network | Accepted | After the technical PoC |
| [ADR-0005](0005-use-go-for-poc-control-plane.md) | 2026-08-17 | Use Go for the PoC Control Plane | Accepted | Before durable multi-host deployment |
| [ADR-0006](0006-loopback-runner-credential-for-poc.md) | 2026-08-18 | Use an identity-bound credential for the loopback Runner PoC | Local PoC only | Before Windows VM transport |
| [ADR-0007](0007-workspace-bound-local-sessions.md) | 2026-08-21 | Use workspace-bound local sessions for the MVP | Local MVP only | Before durable multi-user storage |
| [ADR-0008](0008-store-schedule-instants-in-utc.md) | 2026-08-21 | Store schedule instants in UTC and preserve the submitted zone | Accepted | Before recurring schedules or additional zones |
