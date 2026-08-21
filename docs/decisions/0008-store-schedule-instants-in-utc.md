# ADR-0008: Store Schedule Instants in UTC and Preserve the Submitted Zone

- Status: Accepted
- Date: 2026-08-21

## Context

Operators need to schedule ProofLayer tests in Europe/Istanbul while execution
and audit comparisons require an unambiguous instant.

## Decision

Accept a strict local `YYYY-MM-DDTHH:MM:SS` value with an approved IANA zone,
convert it to UTC for dispatch, and preserve both the original local value and
zone for display and audit. The MVP allowlist is `Europe/Istanbul` and `UTC`.

## Alternatives considered

- Store only the submitted local string: rejected because dispatch order would
  depend on server interpretation and future zone-rule changes.
- Store only UTC: rejected because the operator's intended local wall time and
  zone would be lost from evidence.
- Accept arbitrary IANA zones: deferred to keep the MVP validation surface small.

## Security and data impact

Time-zone names are allowlisted and timestamps are parsed with one strict
layout. No location, customer content, or browser time-zone fingerprint is
collected. Server-side conversion prevents clients from selecting the dispatch
instant independently of the reviewed local value.

## Consequences

Dispatch comparisons are unambiguous and UI output retains operator intent.
Adding zones requires an explicit allowlist change and tests for ambiguous or
nonexistent local times. Durable recurrence and daylight-saving editing are out
of scope for the one-time MVP scheduler.

## Validation

Unit tests cover Europe/Istanbul conversion, invalid zones, past plans,
same-host conflicts, missed plans, and idempotent queue dispatch.
