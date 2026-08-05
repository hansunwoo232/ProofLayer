# Day 5 Completion Report

**Date:** August 2, 2026  
**Status:** Complete. Customer discovery remains deferred under ADR-0002.

## Completed

### Local Splunk lab

- Added a repeatable Docker Compose deployment for Splunk Enterprise.
- Recorded the operator's acceptance of the applicable Splunk General Terms in
  an ignored local environment file.
- Pinned the verified Splunk 10.4.2 AMD64 image digest.
- Bound Splunk Web, HEC, and management endpoints to loopback only.
- Kept the generated self-signed management certificate strictly local and
  recorded certificate hardening as a pre-customer requirement.
- Moved Splunk Web to `http://127.0.0.1:18000` because local port 8000 was
  already occupied; no existing service was stopped.
- Capped the local container at 4 GB memory and 2.5 CPU cores.
- Added persistent volumes, a management-API health check, and safe start/stop
  instructions.
- Generated a strong local administrator password without printing or
  committing it.

### Test index validation

- Created and verified the `prooflayer_test` event index.
- Capped the index at 512 MB for the local proof of concept.
- Sent a synthetic event through Splunk's authenticated simple receiver.
- Verified that a Splunk search returned the ingested event.

```text
Splunk 10.4.2
Container: running healthy
Splunk Web: HTTP 303 redirect to sign-in
Smoke test: PASS
```

### Correlation ID contract

- Finalized the v1 format as `PL-` plus 32 uppercase hexadecimal characters.
- Standardized on 128 bits of cryptographic randomness.
- Updated the test-job and test-run schemas, examples, validator, Windows
  bootstrap, and Sysmon lab filter.
- Defined generation, propagation, exact-match search, expiry, collision,
  redaction, and non-authorization rules.
- Preserved the Day 3 `PL-TEST-BOOTSTRAP-*` value as historical evidence only.

### Risk and assumption tracking

- Started the living risk and assumption log.
- Recorded Apple Silicon emulation, host memory pressure, Splunk licensing,
  time synchronization, offline Windows tooling, marker misuse, and local port
  collision risks.
- Added explicit validation points for the main PoC assumptions.

## Validation

```text
PASS scenario example
PASS test job example
PASS parser-failure run example
PASS cross-contract security invariants
PASS unsafe scenario mutations rejected
PASS Splunk Compose configuration
PASS prooflayer_test index creation and lookup
PASS synthetic event ingest and search
```

## Deferred by decision

Tagging customer interview notes was not performed because ADR-0002 defers all
external outreach until the repeatable MVP is demonstrable. The deferral is
tracked as assumption A-004 in the risk and assumption log.

## Day 6 inputs

- Healthy local Splunk Enterprise instance
- Verified `prooflayer_test` index and repeatable smoke test
- Canonical v1 correlation ID format and lifecycle
- Initial risk and assumption register
- Known-good Windows + Sysmon checkpoint ready for SIEM connectivity work
