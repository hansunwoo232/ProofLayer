# Approved Behavior: Windows Registry Run Key Canary

**Decision:** Approved for isolated-lab implementation and verification
**Date:** August 11, 2026
**Owner:** Founding team
**Scenario:** `windows-registry-run-key-canary` version `0.1.0`

## Approved behavior

The built-in handler creates exactly one value under
`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`. The value name is
`ProofLayer_` plus the 128-bit canonical correlation identifier. Its data is a
fixed harmless command that exits immediately. The handler then removes that
exact value and queries the registry independently to prove it is absent.

The Registry key itself is never deleted or modified beyond the dedicated
canary value.

## Safety boundary

- Fixed HKCU path; no HKLM or remote registry access
- No caller-defined key, value name, value data, command, or argument
- Canary name derived only from a validated canonical correlation ID
- No privilege elevation or administrator requirement
- No network access, customer data access, or credential access
- Cleanup runs after every execution outcome, including cancellation and error
- Cleanup receives its own bounded context and does not inherit cancellation
- Artifact absence is verified separately after deletion
- A remaining value produces terminal `artifact_remaining`, never PASS

## Expected telemetry

- Provider: `Microsoft-Windows-Sysmon`
- Event ID: `13` (`Registry value set`)
- Required fields: `host.name`, `registry.path`, `registry.value`, `user.name`
- Correlation: exact canonical `PL-` ID encoded in the canary value name

## Approval limits

Approval applies only to the isolated ProofLayer lab and the fixed Windows API
handler. It does not approve arbitrary registry mutation, customer production
execution, a configurable value payload, or persistence testing that survives
the scenario cleanup window.

## Revocation conditions

Approval is revoked if any path or value data becomes caller-controlled, the
handler targets HKLM, cleanup can be skipped, absence is inferred rather than
queried, or sensitive values are returned in logs or reports.
