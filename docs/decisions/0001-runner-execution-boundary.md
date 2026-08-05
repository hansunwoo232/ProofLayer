# ADR-0001 — The Runner will not execute arbitrary commands

- **Date:** 2026-07-28
- **Status:** Accepted
- **Owner:** Founding team

## Context

The Runner creates test behavior on customer Windows hosts. Compromise or misuse
could otherwise turn the product into a remote attack tool. Control-plane
authorization alone is not a sufficient boundary.

## Decision

The Runner will not accept general shell or PowerShell text. It executes only
jobs containing a known scenario ID, a versioned narrow parameter schema,
host/scope binding, expiry, a one-time job ID, and mandatory cleanup. It rejects
unknown scenarios and extra arguments locally even if the control plane is
assumed compromised.

## Alternatives

1. Send command text from the control plane: fast, but unacceptable risk.
2. Server-side allowlist only: does not survive control-plane compromise.
3. Runner-side allowlist plus signed packages: selected.

## Security and data impact

- The control-plane-to-Runner path is the highest-risk trust boundary.
- The scenario signing key must remain separate from runtime systems.
- Negative test: an extra argument invalidates an otherwise valid signed job.
- Negative test: an unknown scenario ID never falls through to a shell.

## Consequences

- Positive: materially smaller attack surface and blast radius.
- Negative: custom scenario development and release are slower.
- Reversal condition: none; general command execution violates the product
  safety principle.

## Validation

Validate with Runner prototype negative tests and an independent security review
before the first pilot.
