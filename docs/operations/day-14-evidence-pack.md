# Day 14 Evidence Pack

**Prepared:** August 5, 2026  
**Scope:** Technical PoC evidence only

## Evidence inventory

| Claim | Evidence | Result |
|---|---|---|
| Safe Windows behavior produces endpoint telemetry | Three Sysmon Event ID 1 records with unique correlation IDs | PASS |
| Splunk receives the exact events | Three exact-ID results in `prooflayer_test` | PASS |
| Required fields survive ingestion | Four of four fields present in every run | PASS |
| Result is repeatable | Three consecutive unique runs | PASS |
| Observer is least privilege | Target index allowed; `_internal` denied | PASS |
| Minimum-metadata boundary is enforced | Reports contain field names and counts, not field values | PASS |
| Detection rule fires | No detection connector yet | NOT TESTED |
| Alert is created | No alert connector yet | NOT TESTED |
| Visible product-stage result | No Runner or dashboard implementation yet | NOT TESTED |
| Customer demand | Outreach deferred under ADR-0002 | NOT EVALUATED |

## Repeatability measurements

| Correlation ID | Sysmon record | Splunk ingestion latency |
|---|---:|---:|
| `PL-1188F1C905964BB7863D97CE8E6BBEB8` | 2447 | 2677 ms |
| `PL-4022D6E44B3A48A6AC36493644C820A3` | 2481 | 1679 ms |
| `PL-79E45D99A1AD449AB9C079F1C700E43A` | 2484 | 1365 ms |

Minimum / average / maximum Splunk ingestion latency: **1365 / 1907 / 2677
ms**.

## Reproduction commands

From `deployments/splunk`, with ignored local credentials provisioned:

```text
./verify-windows-event-by-id.sh PL-1188F1C905964BB7863D97CE8E6BBEB8
./verify-windows-event-series.sh
./verify-observer-access.sh
```

Machine-local JSON and screenshots remain ignored under
`work/prooflayer-lab/evidence`. This document is the redacted, commit-safe
summary.
