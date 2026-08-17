# ADR-0002 — Defer Customer Outreach Until the MVP Is Demonstrable

- **Date:** 2026-07-29
- **Status:** Accepted
- **Owner:** Founding team

## Context

The first five company-level outreach messages are drafted, but the product does
not yet have a demonstrable end-to-end MVP. Early interviews could validate the
problem sooner, while premature outreach may spend high-value contacts before
the team can show credible evidence.

## Decision

Customer outreach is deferred until the MVP can demonstrate:

1. A safe scenario executed on an authorized Windows lab host.
2. A unique correlation ID in endpoint telemetry.
3. Splunk ingestion and required-field validation.
4. A visible stage-level PASS/FAIL result.
5. Reliable cleanup.

The existing Barikat, Cyberwise, Biznet Bilişim, ADEO Cyber Security, and DnDx
drafts remain preserved but will not be sent before this gate.

## Consequences

- Positive: first contact includes a concrete product demonstration.
- Positive: security and product claims can be supported with evidence.
- Negative: problem discovery and design-partner learning begin later.
- Mitigation: keep the interview guide, account list, and drafts current while
  the MVP is being built.

## Revisit condition

Revisit immediately after the first repeatable Windows-to-Splunk end-to-end run,
or earlier if a warm inbound design-partner opportunity appears.

## August 11, 2026 review

The repeatable technical run and reliable cleanup demonstration now exist. The
Day 24 Registry canary returned cleanup PASS and an independent artifact-absence
query returned PASS. The visible product-facing stage view is still not
implemented, so the instruction to contact prospects only after the MVP remains
in force. Review again when all five decision-gate items can be shown in one
product-facing demo.
