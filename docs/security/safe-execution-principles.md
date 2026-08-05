# Safe Execution Principles

These principles are mandatory for every ProofLayer Runner, scenario, lab, and
demo implementation.

## Allowed purpose

ProofLayer may generate only deterministic, reversible, synthetic behavior that
exists to validate telemetry and detection controls in an explicitly authorized
environment.

## Non-negotiable prohibitions

- No real credential collection, dumping, replay, guessing, or validation.
- No exploit execution or vulnerability exploitation.
- No arbitrary shell, script, command, binary, URL, or payload supplied by a
  control plane operator.
- No ransomware behavior, encryption, destructive deletion, log clearing, or
  security-control disabling.
- No collection or exfiltration of customer files, event bodies, secrets,
  tokens, browser data, or personal data.
- No internet access during scenario execution.
- No persistence that survives required cleanup.
- No lateral movement, privilege escalation, or remote execution in the MVP.
- No execution outside a registered host, approved maintenance window, signed
  job, and allowlisted scenario version.

## Required controls

1. A scenario selects a fixed built-in handler; it never carries a command.
2. Parameters are typed, bounded, schema-validated, and handler-specific.
3. Every job is signed, short-lived, bound to one host, and replay-protected.
4. The Runner enforces time, CPU, memory, network, and output limits locally.
5. Cleanup runs even after execution or observation failure.
6. Cleanup verifies artifact absence and reports failure as a terminal error.
7. A local kill switch prevents new work and stops safe in-flight work.
8. Evidence contains minimum metadata and hashes, not raw sensitive events.
9. Correlation IDs are opaque identifiers, not authorization credentials.
10. Every new handler requires threat-model review and negative tests.

## Lab-only exceptions

The local Day 6 Windows-to-Splunk proof permits a self-signed HEC certificate
only for the exact guest-visible address `10.0.2.100:8088`. The VM remains under
QEMU `restrict=on`, and the command-backed forwarding rule starts only
`/usr/bin/nc 127.0.0.1 8088` for that fixed guest destination. This exception is
forbidden in customer or shared environments.

The read-only Day 6 ISO contains a dedicated local HEC token. The ISO is ignored
by Git, must not be shared, and the token must be rotated when the experiment is
retired.
