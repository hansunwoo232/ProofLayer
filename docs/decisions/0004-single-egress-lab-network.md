# ADR-0004 — Use a Single-Egress Windows-to-HEC Lab Network

- **Date:** 2026-08-02
- **Status:** Accepted
- **Owner:** Founding team

## Context

The Windows guest must submit one synthetic event to host Splunk without gaining
general host or internet access.

## Decision

Add a QEMU `siem` network mode that retains `restrict=on`. A command-backed
`guestfwd` rule accepts only guest `10.0.2.100:8088` and starts the fixed host
relay `/usr/bin/nc 127.0.0.1 8088`. No other guest traffic is routed to the host
or outside network.

## Consequences

- The proof can measure real endpoint-to-SIEM ingestion latency.
- Splunk remains loopback-only on the host.
- The Windows script requires a lab-only self-signed certificate exception for
  the exact guest-visible HEC address.
- This design is a local PoC boundary and not a customer network architecture.

## Validation

Validated on August 5, 2026:

- HEC health returned code 17 (`HEC is healthy`) from the guest.
- Three authenticated events were accepted and indexed.
- QEMU `restrict=on` remained enabled.
- Splunk ports remained bound to host loopback.
