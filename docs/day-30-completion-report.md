# Day 30 Completion Report

**Date:** August 19, 2026

**Status:** Complete; the end-to-end technical PoC gate passed in the Windows lab.

## Gate

One approved operator action must show Runner execution, Windows telemetry,
Splunk ingestion, field validation, detection outcome, and mandatory cleanup.
Alert delivery may remain `not_tested` because it is outside the Day 30 PoC.

## Implemented

- One-job Windows worker orchestration across all seven canonical stages
- Signed job verification before execution
- Fixed process-marker-only Day 30 harness with no runtime arguments
- Local Sysmon Event ID 1 observation
- Bounded Unicode XML collection through `wevtutil /uni:true`
- BOM-aware and BOM-less UTF-16LE/UTF-16BE event decoding
- Record-isolated XML parsing so unrelated malformed records fail closed without
  hiding a valid bounded correlation event
- Stable endpoint failure classifications with detailed diagnostics kept local
- Bounded HEC export containing only synthetic canary values and event metadata
- Exact Splunk polling, four-field validation, and inline detection search
- Mandatory cleanup publication on success and failure paths
- Optional alert `not_tested` state after successful detection
- Stable local signing seed support
- Separate loopback browser and isolated TLS Runner listeners
- Fixed QEMU forwards for HEC, Splunk management, and Runner TLS
- Ignored secret/certificate preparation and Windows Arm64 ISO packaging

## Live acceptance result

The live Windows/QEMU run completed successfully.

| Evidence | Value |
|---|---|
| Job ID | `d6ab8e1c-9350-4c7f-bd81-e566e4e98eb8` |
| Correlation ID | `PL-07A96D50873BDB451BBB69DFD26A97C6` |
| Terminal status | `completed` |
| Execution | PASS · 297 ms |
| Endpoint telemetry | PASS · 1,495 ms |
| SIEM ingestion | PASS · 1,376 ms |
| Field validation | PASS |
| Detection | PASS · 87 ms |
| Alert | `not_tested` · expected PoC boundary |
| Cleanup | PASS |

The bounded status response is stored in
[`docs/evidence/day-30-live-acceptance.json`](evidence/day-30-live-acceptance.json),
and the Windows Runner terminal evidence is stored in
[`docs/evidence/day-30-live-acceptance.png`](evidence/day-30-live-acceptance.png).
Neither artifact contains a credential, raw endpoint event, or customer data.

## Defect resolved during acceptance

The initial Windows runs proved process execution but failed local endpoint XML
decoding. The failure was isolated to the Windows event output encoding rather
than Sysmon publication, query lookback, evidence size, or the SIEM path.

The Runner now requests Unicode output explicitly and accepts bounded UTF-16
event streams with or without a byte-order mark. Detailed XML diagnostics are
shown only by the local lab harness; the Control Plane retains the stable,
non-sensitive `endpoint_xml_invalid` classification.

## Automated evidence

- Full Control Plane unit and lifecycle test suite
- Full Runner unit, HEC boundary, observer, parser, and worker test suites
- UTF-8, UTF-16LE, UTF-16BE, BOM, BOM-less, declaration, malformed-record, and
  invalid-timestamp parser coverage
- Go race detector and static analysis
- Windows Arm64 cross-compilation of the Day 30 harness
- Generated accepted Runner ISO SHA-256:
  `879411b955ea0e9c812eaf4e4c967b687f4d62f72b975a01c1a6679f2c8b17ad`

## Commercial work

Customer demos and follow-up remain intentionally deferred under ADR-0002, as
requested by the founder. The product will not be presented externally until
the technical MVP is repeatable beyond this PoC gate.
