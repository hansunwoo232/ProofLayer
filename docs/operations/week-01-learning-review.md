# Week 1 Learning Review

## Evidence-backed technical learning

- Windows 11 Pro Arm64 runs reliably under QEMU/HVF on the Apple Silicon host.
- Sysmon v15.21 Arm64 produces deterministic process-creation evidence for a
  harmless correlation marker.
- Splunk 10.4.2 runs under AMD64 emulation, but initial provisioning is slow and
  readiness must include the Ansible completion gate.
- A health check that accepts any management API `401` can report readiness
  before provisioning is complete; the lab now requires both API health and the
  provisioning completion marker.
- Empty local secret values must fail before container startup. The setup script
  now replaces invalid local environment files and preserves the old copy.
- Splunk's internal password state must never be edited in place. Affected
  volumes were preserved and fresh explicitly named volumes were used.
- QEMU `restrict=on` plus a fixed command-backed `guestfwd` rule provides a
  narrow Windows-to-HEC path without general host or internet access. Direct
  TCP forwarding accepted connections but did not relay payload bytes to the
  Docker-published loopback port on this host.
- Windows 11 Arm64 required a Microsoft-signed Arm64 NetKVM driver before the
  VirtIO adapter could use the HEC relay.
- Three end-to-end runs produced 1365–2677 ms Splunk ingestion latency, with a
  1907 ms average.
- Splunk JSON extraction can create duplicate multivalue fields when `spath`
  runs over fields already extracted by the sourcetype; observer queries must
  normalize with `mvindex` before exact validation.

## Product assumptions

- The core endpoint-to-SIEM reliability problem remains the working product
  hypothesis.
- Customer-problem frequency, willingness to pay, and pilot timing remain
  unvalidated because outreach is intentionally deferred under ADR-0002.
- No customer discovery claim may be inferred from technical lab success.

## Scope decisions

- Continue with one harmless Windows process-marker scenario.
- Continue with Splunk only.
- Defer detection and alert stages until endpoint telemetry ingestion is
  repeatable.
- Defer brand selection, customer messaging, and outreach until the MVP demo
  gate specified by ADR-0002.

## Next validation questions

1. Windows event-to-index latency is repeatable across three runs: **yes**.
2. A least-privilege observer can search only `prooflayer_test`: **yes**.
3. The scenario contract rejects the current arbitrary-command mutations:
   **yes**.
4. A second operator can reproduce the lab from the runbook: **not yet
   independently tested**.
