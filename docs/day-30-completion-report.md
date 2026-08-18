# Day 30 Completion Report

**Date:** August 18, 2026

**Gate:** One operator action shows Runner execution, Windows telemetry, Splunk
ingestion, field validation, and detection outcome.

## Implemented

- One-job Windows worker orchestration across all seven canonical stages
- Signed job verification before execution
- Fixed process-marker-only Day 30 harness with no runtime arguments
- Local Sysmon Event ID 1 observation
- Bounded HEC export containing only synthetic canary values and event metadata
- Exact Splunk polling, four-field validation, and inline detection search
- Mandatory cleanup publication on success and failure paths
- Optional alert `not_tested` state after successful detection
- Stable local signing seed support
- Separate loopback browser and isolated TLS Runner listeners
- Fixed QEMU forwards for HEC, Splunk management, and Runner TLS
- Ignored secret/certificate preparation and Windows Arm64 ISO packaging

## Automated evidence

- Control Plane unit and lifecycle tests
- Runner unit, HEC security-boundary, and worker orchestration tests
- Windows Arm64 cross-compilation of the Day 30 harness
- Shell syntax validation
- Authenticated TLS lease smoke test returned HTTP 204
- Generated ignored Runner ISO SHA-256:
  `86c8a647ce698bafb29b9afad2eda7334ca0a46192f0c2bc1c449d640c3b9fbe`

## Live acceptance evidence

Pending the terminal Windows run for correlation `PL-C00439FF…8530CF`.
Do not mark the Day 30 gate PASS until the browser shows a terminal result and
the evidence screenshot or JSON is stored in the ignored lab evidence folder.

## Commercial work

Customer demos and follow-up remain intentionally deferred under ADR-0002, as
requested by the founder. The product will not be presented externally until
the technical MVP is repeatable.
