# ADR-0003 — Separate Splunk Ingestion and Observation Identities

- **Date:** 2026-08-02
- **Status:** Accepted
- **Owner:** Founding team

## Context

The Windows test host must submit synthetic telemetry, while the future Observer
must search for exact correlation evidence. Sharing the Splunk administrator
credential or one bidirectional token would expand the blast radius.

## Decision

Use three identities:

1. `admin` for local provisioning only.
2. A dedicated `prooflayer_lab` HEC token restricted to `prooflayer_test` for
   ingestion.
3. A `prooflayer_observer` user with only the `search` capability and only the
   `prooflayer_test` index for runtime observation.

## Consequences

- The Runner cannot search Splunk or administer the SIEM.
- The Observer cannot ingest events or administer Splunk.
- Local setup requires separate secret generation and negative authorization
  tests.
- Customer deployment must replace local files and self-signed TLS with managed
  secrets and trusted certificates.

## Validation

`configure-observer-role.sh` proves that the observer can search
`prooflayer_test` and cannot return `_internal` events.
