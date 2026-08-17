# Risk and Assumption Log

This is a living engineering and product record. Review it at each weekly gate
and whenever a risk changes status.

## Risks

| ID | Risk | Likelihood | Impact | Mitigation / trigger | Owner | Status |
|---|---|---:|---:|---|---|---|
| R-001 | The official Splunk container is AMD64-only and runs through emulation on the Apple Silicon development host. | High | Medium | Pin the image, keep the lab workload small, measure latency, and move to an x86-64 host if emulation blocks repeatable tests. | Engineering | Open |
| R-002 | The 16 GB development host may experience memory pressure when Windows QEMU and Splunk run together. | High | Medium | Stop unused workloads, cap Splunk at 4 GB, and reduce Windows guest memory before simultaneous testing if needed. | Engineering | Open |
| R-003 | Splunk licensing or terms could prevent redistribution or long-running use of the local evaluation environment. | Medium | High | Use the official image only for an operator-accepted local lab; do not redistribute credentials, images, or licensed binaries. Reassess before any public or customer deployment. | Founder | Open |
| R-004 | Clock drift or non-UTC systems can produce false ingestion and latency results. | Medium | High | Keep the Windows bootstrap UTC guard, record UTC timestamps, and add explicit skew measurement before cross-system correlation. | Engineering | Mitigating |
| R-005 | Offline Windows setup makes repeatable driver and tool delivery fragile. | Medium | Medium | Preserve the tools ISO manifest and hashes; rebuild from verified inputs; bundle only SHA-256-pinned Microsoft-signed Arm64 NetKVM files. | Engineering | Mitigated |
| R-006 | A correlation marker could be mistaken for authorization or expose customer context in reports. | Low | High | Keep IDs opaque, random, non-semantic, signed inside jobs, and redacted in public reports. | Security | Mitigating |
| R-007 | Common development ports may already be occupied by local services. | Medium | Low | Bind Splunk Web to loopback port 18000, keep management ports loopback-only, and verify availability before startup. | Engineering | Mitigated |
| R-008 | The local Splunk management endpoint uses its generated self-signed certificate and CLI hostname validation is disabled. | High | Medium | Keep the endpoint loopback-only for the PoC. Require trusted certificates and hostname verification before any shared or customer environment. | Security | Mitigating |
| R-009 | In-place manipulation of Splunk's internal password files can leave the 10.4 sidecar-based container with no local users. | Low | Medium | Never edit the password database in place. Preserve affected volumes, rotate to fresh explicitly named lab volumes, and recreate synthetic-only test data. | Engineering | Mitigated |
| R-010 | QEMU's direct TCP `guestfwd` can accept the guest connection without relaying payload bytes to a Docker-published loopback port on this macOS host. | Medium | Medium | Use QEMU's command-backed `guestfwd` with fixed `/usr/bin/nc 127.0.0.1 8088`; keep `restrict=on`; verify HEC health before every proof. | Engineering | Mitigated |
| R-011 | The local Control Plane queue, lifecycle state, and signing key are lost on restart. | High | Medium | Treat the process as a local PoC, show interrupted runs as unavailable, and add durable state plus managed key storage before a shared environment. | Engineering | Open |

## Assumptions

| ID | Assumption | Validation method | Review point | Status |
|---|---|---|---|---|
| A-001 | AMD64 emulation is fast enough for the first endpoint-to-Splunk proof of concept. | Three runs produced Splunk ingestion latency of 1365–2677 ms, averaging 1907 ms. | Day 7 | Validated |
| A-002 | A 512 MB `prooflayer_test` index is sufficient for local synthetic telemetry. | Monitor index size and retention during the first six scenarios. | Day 30 | Unvalidated |
| A-003 | A five-minute pre-window and ten-minute post-window are sufficient for initial correlation. | Three observed Sysmon-to-Splunk delays were below three seconds; retain bounded clock-skew checks before customer use. | Day 7 | Validated for local PoC |
| A-004 | Deferring external outreach until a demonstrable MVP will not remove access to the first design partners. | Resume the interview/outreach plan immediately after the MVP demo gate. | MVP demo gate | Accepted |

## Decision references

- `docs/decisions/0002-defer-outreach-until-mvp.md`
- `docs/security/threat-model-v0.1.md`
- `docs/architecture/correlation-id-lifecycle.md`
