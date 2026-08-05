# Correlation ID Lifecycle v1

## Purpose

One correlation ID ties together a ProofLayer test job, the harmless endpoint
behavior it produces, endpoint telemetry, SIEM events, detection results, and
the final test-run record. It is an opaque test identifier, not an authorization
token.

## Canonical format

```text
PL-7F9A1C3D5E8B2A104C6D9F013B7E52A8
```

The v1 format is exactly `PL-` followed by 32 uppercase hexadecimal characters:

```regex
^PL-[A-F0-9]{32}$
```

The suffix is 16 cryptographically random bytes rendered as uppercase hex.
Generators must use an operating-system cryptographic random source. UUID v4 is
acceptable only when the implementation preserves all 32 hexadecimal digits and
removes separators.

## Lifecycle

1. The control plane generates the ID while creating a signed test job.
2. The ID becomes part of the signed job payload.
3. The Runner validates the job before placing the ID in an allowlisted marker.
4. Endpoint telemetry records the marker.
5. The Observer searches only the authorized environment and time window for the
   exact ID.
6. Every stage result references the same ID.
7. The completed run retains the ID for audit and regression history.

## Invariants

- Generate exactly one ID per test run; never recycle or retry with the same ID.
- Treat an exact-match collision as a security-relevant error and abort the run.
- Do not encode tenant, host, user, scenario, timestamp, or other customer data.
- Do not accept lowercase, separators, whitespace, or user-supplied prefixes.
- Do not use the ID as proof of identity, permission, or message authenticity.
- Logs may retain the ID, but public reports must use a redacted form such as
  `PL-7F9A1C…E52A8`.

## Search window

The Observer derives its search window from the signed job and measured clock
skew. The initial PoC window is five minutes before execution through ten
minutes after execution. A timeout closes observation; it does not permit ID
reuse.

## Historical lab marker

The Day 3 bootstrap used `PL-TEST-BOOTSTRAP-7E31F075A720` before this contract
was finalized. It remains valid historical evidence but is not a v1 production
correlation ID. Future bootstrap runs use the canonical v1 format.
