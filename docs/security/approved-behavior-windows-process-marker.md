# Approved Behavior: Windows Process Marker

**Decision:** Approved for the isolated technical PoC  
**Date:** August 5, 2026  
**Owner:** Founding team  
**Scenario:** `windows-process-marker` version `0.1.0`

## Approved behavior

The scenario generates a cryptographically random canonical correlation ID and
passes it to one fixed built-in Windows process-marker handler. The handler
creates a short-lived, harmless process whose command line contains that ID and
whose output is discarded. The behavior exists only to produce a Sysmon Process
Create event.

## Safety boundary

- No scenario-defined executable, command, argument list, or shell fragment
- No credential access, privilege escalation, persistence, or evasion
- No customer file, registry, service, task, account, or network mutation
- No outbound connection from the scenario behavior
- No user-supplied payload; the only variable value is the generated ID
- Hard runtime deadline and cancellation support
- Host authorization, maintenance-window, and kill-switch checks before launch
- Exact built-in handler version resolved from the local signed catalog
- Artifact-absence verification after every outcome

## Expected telemetry

- Provider: `Microsoft-Windows-Sysmon`
- Event ID: `1`
- Required fields: `host.name`, `process.name`, `process.command_line`,
  `user.name`
- Correlation: exact canonical `PL-` ID

## Approval limits

This approval applies only to the isolated ProofLayer lab and the fixed handler
described above. It does not approve arbitrary process execution, a generic
shell interface, customer production execution, or any new scenario action.
Material behavior changes require a new security review and version.

## Revocation conditions

Approval is revoked if the implementation accepts arbitrary executable or
argument input, leaves a persistent artifact, exceeds its deadline, bypasses a
preflight control, emits secret material, or cannot prove cleanup.
