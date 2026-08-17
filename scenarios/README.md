# Scenarios

Scenario definitions are versioned, signable, and machine-verifiable. Each
scenario declares a fixed built-in handler, network scope, expected telemetry,
time limit, and mandatory cleanup verification.

The current built-in definitions are:

- `windows-process-marker.yaml`: emits a harmless process command-line marker.
- `windows-registry-run-key-canary.yaml`: creates one correlation-bound HKCU Run
  value, immediately removes it, and independently verifies absence.
- `windows-scheduled-task-canary.yaml`: creates one correlation-bound, harmless
  logon task, immediately deletes it, and independently verifies absence.

YAML files are parsed and validated against `schemas/v1/scenario.schema.json`
before the Runner resolves the local built-in handler.
