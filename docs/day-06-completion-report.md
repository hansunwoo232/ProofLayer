# Day 6 Completion Report

**Date:** August 5, 2026  
**Status:** Complete. Customer outreach remains deferred under ADR-0002.

## Completed

- Executed one deterministic and harmless `cmd.exe` correlation-marker action
  on the authorized Windows 11 Arm64 lab host.
- Generated the canonical correlation ID
  `PL-1188F1C905964BB7863D97CE8E6BBEB8`.
- Found the exact marker in Sysmon Event ID 1, record ID 2447.
- Validated endpoint fields for UTC time, image, command line, and user.
- Submitted a minimum structured JSON event through the dedicated local HEC
  token without exposing Splunk administrator or observer credentials.
- Received HEC acceptance in 2196 ms from the endpoint event timestamp.
- Preserved a local ignored screenshot and JSON search evidence.

## Lab security improvements

- Installed the Microsoft-signed Windows 11 Arm64 NetKVM driver from a
  SHA-256-pinned VirtIO-Win package.
- Kept Splunk bound to host loopback.
- Retained QEMU `restrict=on` and exposed only guest
  `10.0.2.100:8088` through a fixed command-backed relay to local HEC.
- Disabled system proxy use in the lab-only HEC client.
- Kept the self-signed certificate exception bound to the exact lab HEC URI.

## Acceptance result

```text
Safe execution:                  PASS
Canonical correlation ID:       PASS
Sysmon Event ID 1:               PASS
Required endpoint fields:        PASS
HEC authenticated acceptance:    PASS
General guest internet access:   DISALLOWED BY LAB MODE
```

## Deferred by decision

Scheduling three customer interviews was not performed because ADR-0002
preserves the founder's instruction to contact prospects only after the MVP is
demonstrable. This is a deferral, not completed discovery work.
