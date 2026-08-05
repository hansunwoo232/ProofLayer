# Scenarios

Scenario definitions are versioned, signable, and machine-verifiable. Each
scenario declares a fixed built-in handler, network scope, expected telemetry,
time limit, and mandatory cleanup verification.

`windows-process-marker.yaml` is the first safe scenario definition. YAML files
are parsed and validated against `schemas/v1/scenario.schema.json` before the
Runner resolves the local built-in handler.
