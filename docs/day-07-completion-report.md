# Day 7 Completion Report

**Date:** August 5, 2026  
**Status:** Complete. Outreach conversion work remains deferred under ADR-0002.

## Repeatability proof

The same safe Windows → Sysmon → HEC → Splunk path passed three times with
three unique canonical correlation IDs.

| Correlation ID | Sysmon record | Endpoint-to-HEC accepted | Splunk ingestion |
|---|---:|---:|---:|
| `PL-1188F1C905964BB7863D97CE8E6BBEB8` | 2447 | 2196 ms | 2677 ms |
| `PL-4022D6E44B3A48A6AC36493644C820A3` | 2481 | 1231 ms | 1679 ms |
| `PL-79E45D99A1AD449AB9C079F1C700E43A` | 2484 | 1139 ms | 1365 ms |

Endpoint-to-HEC acceptance averaged 1522 ms. Splunk ingestion latency was:

- Minimum: 1365 ms
- Average: 1907 ms
- Maximum: 2677 ms

## Independent Splunk validation

The least-privilege `prooflayer_observer` account verified all three events in
`prooflayer_test`. Every result contained:

- Sysmon Event ID 1
- Exact correlation ID
- `host.name`
- `process.name`
- `process.command_line`
- `user.name`

The runtime observer did not receive the HEC token or Splunk administrator
credential. Its negative authorization test continued to prevent `_internal`
event access.

## Evidence

Local evidence is deliberately excluded from Git:

```text
work/prooflayer-lab/evidence/day-06-windows-pass.png
work/prooflayer-lab/evidence/day-07-repeatability-pass.png
work/prooflayer-lab/evidence/day-07-latest-splunk-result.json
work/prooflayer-lab/evidence/day-07-splunk-series.json
```

## Week 1 conclusion

The endpoint-to-SIEM technical hypothesis is feasible on the development host.
This result does not validate customer frequency, willingness to pay, or pilot
timing.
