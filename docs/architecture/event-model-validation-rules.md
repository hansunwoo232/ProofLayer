# Event Model Validation Rules

## Versioning

- Every payload includes `schema_version`.
- Version `1.0` rejects unknown top-level fields.
- Breaking changes require a new schema version and migration note.

## Identity

- Job and run IDs are UUIDs.
- Correlation IDs match `^PL-[A-F0-9]{32}$` exactly.
- Job and run correlation IDs must match.
- Scenario ID and version must match the signed job.

## Time

- All contract timestamps are RFC 3339 UTC.
- `requested_at < expires_at`.
- `started_at <= completed_at` when completion exists.
- Stage completion cannot precede stage start.
- Latency values are non-negative integers.

## Stage behavior

- Supported order is execution, endpoint telemetry, SIEM ingestion, field
  validation, detection, alert, and cleanup.
- A failed required upstream stage makes dependent stages `not_tested`.
- Cleanup remains required even when an earlier stage fails.
- `passed` requires all required stages and cleanup to pass.
- Raw customer event bodies are excluded from stage evidence.

## Evidence

- Evidence contains counts, identifiers, matched/missing field names, index,
  sourcetype, and hashes only.
- Hashes are lowercase SHA-256 hexadecimal.
- Error messages are bounded and must not contain secrets or raw events.

## Executable checks

The repository's dependency-free contract suite validates the JSON examples and
cross-contract invariants:

```bash
node tools/validate-contract-examples.mjs
```

Full JSON Schema Draft 2020-12 validation will be added to CI when the first
runtime package introduces a pinned schema dependency.
