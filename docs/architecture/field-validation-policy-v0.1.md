# Field Validation Policy v0.1

**Status:** Frozen for `windows-process-marker@0.1.0`  
**Date:** August 5, 2026

## Required fields

- `host.name`
- `process.name`
- `process.command_line`
- `user.name`

The Splunk connector evaluates field presence inside the Splunk boundary and
returns four boolean flags. It never returns the values. The validator accepts
only those four field names; unknown evidence keys fail closed.

## Decisions

- Four present fields: `passed`
- Any false or absent field: `failed` with the exact missing field names
- Unknown evidence field: invalid evidence; no stage PASS
- Field-validation failure: `detection` and `alert` become `not_tested`

The result contains counts and field names only. Its typed API cannot accept a
command line, username, hostname, raw event, or arbitrary event property.

## Parser-failure fixture

`docs/examples/day-21-missing-command-line-result.json` demonstrates a matching
event where `process.command_line` is absent. It produces a stable
`missing_required_field` root cause without exposing an endpoint value.
