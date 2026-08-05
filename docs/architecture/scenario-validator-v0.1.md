# Scenario Validator Design v0.1

## Goal

Reject unsafe or malformed scenario definitions before packaging, signing, or
Runner execution. Validation is fail-closed and deterministic.

## Proposed interface

```text
prooflayer validate scenario <path> [--format json]
```

Success writes one machine-readable result and exits `0`. Failure writes stable
error codes and exits non-zero. The validator never executes a handler.

## Validation pipeline

1. Read a bounded local file; reject links and files over 256 KB.
2. Parse YAML with duplicate-key rejection and aliases disabled, then normalize
   to JSON.
3. Validate against JSON Schema Draft 2020-12.
4. Apply semantic security invariants that cannot be expressed clearly in the
   schema.
5. Emit a canonical JSON representation for hashing and signing.

## Mandatory semantic checks

- `action` maps to exactly one compiled built-in `handler`.
- No unknown property survives normalization.
- No command, shell, interpreter, URL, raw arguments, or executable path field
  exists anywhere in the document.
- `network_access` is `none` for every MVP action.
- Cleanup is required, bounded, handler-compatible, and verifies absence.
- Timeout and retry values stay within product-wide maxima.
- Required telemetry fields come from the supported field registry.
- Scenario ID and version are immutable package identity inputs.
- Parameter schema is bounded and cannot introduce object, array, or free-form
  string payloads unless the handler explicitly supports them.

## Stable errors

| Code | Meaning |
|---|---|
| `SCENARIO_PARSE_ERROR` | Input is not valid restricted YAML/JSON |
| `SCENARIO_SCHEMA_ERROR` | JSON Schema validation failed |
| `UNKNOWN_ACTION` | Action is not compiled into the Runner |
| `HANDLER_MISMATCH` | Action does not map to the declared handler |
| `ARBITRARY_EXECUTION_FIELD` | A command-like field is present |
| `NETWORK_POLICY_VIOLATION` | Scenario requests unsupported network access |
| `CLEANUP_POLICY_VIOLATION` | Cleanup is absent, unbounded, or incompatible |
| `TELEMETRY_FIELD_UNSUPPORTED` | Expected field is not in the registry |

## Current bridge

`tools/validate-contract-examples.mjs` already checks the first scenario example
and rejects unsafe mutations for handler mismatch, network access, missing
cleanup, missing absence verification, and unknown properties. The production
validator will move these invariants into the Go Runner toolchain without
weakening the existing tests.

## Acceptance tests

- Known-good process marker passes.
- Every single prohibited field mutation fails.
- Unknown action and handler mismatch fail.
- Cleanup removal and cleanup relaxation fail.
- Duplicate YAML keys and aliases fail.
- Canonical output is byte-identical across repeated validation.
- Validation never starts a process or opens a network connection.
