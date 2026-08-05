# ProofLayer

ProofLayer is the working codename for a **Detection Reliability Platform**
that verifies whether the path from safe synthetic endpoint behavior to SIEM
detection and alerting actually works.

> Brand note: “ProofLayer” is an internal codename only. The July 27, 2026
> screening found active products using the same name. Do not use it as the
> customer-facing brand or purchase a domain under this name.

## 90-day objective

By the end of July 27–October 24, 2026:

- The Windows Runner executes six safe scenarios.
- Sysmon and Windows Event Log telemetry is verified.
- Splunk ingestion, required fields, detection, and alert outcomes are checked.
- Results are shown as stage-level PASS, FAIL, or NOT TESTED.
- At least one design-partner pilot and written pilot intent are obtained.
- A synthetic public demo that does not connect to real systems is ready.

See [90-day-success-criteria.md](docs/product/90-day-success-criteria.md) for
success criteria and [mvp-scope.md](docs/product/mvp-scope.md) for scope.

## Monorepo structure

```text
prooflayer/
├── runner/          Safe Windows scenario executor
├── control-plane/   API, scheduler, correlation, and audit
├── observer/        Splunk and endpoint validation adapters
├── scenarios/       Signable, allowlisted safe scenario definitions
├── dashboard/       Operator interface
├── deployments/     Local lab and customer deployment packages
└── docs/            Product, architecture, security, and discovery records
```

## Initial engineering principles

1. No real exploitation, credential dumping, or data exfiltration.
2. The Runner never executes arbitrary commands; it accepts only signed,
   allowlisted scenarios.
3. Every scenario defines a timeout, expected telemetry, and cleanup behavior.
4. Only minimum test metadata may leave a customer environment.
5. A feature is not complete without test and security acceptance criteria.
6. The single Windows → Splunk path comes before broad platform features.

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution and quality rules.

## Current working documents

- [One-page product brief](docs/product/one-page-product-brief.md)
- [Customer interview guide](docs/customer-discovery/interview-guide.md)
- [Threat model v0.1](docs/security/threat-model-v0.1.md)
- [Weekly operating cadence](docs/operations/weekly-cadence.md)
- [Decision log](docs/decisions/decision-log.md)
- [Project language policy](docs/operations/language-policy.md)

## Current validated proof

- Three unique Windows → Sysmon → HEC → Splunk runs passed.
- Required endpoint fields were parsed in Splunk.
- Splunk ingestion latency was 1365–2677 ms, averaging 1907 ms.
- The runtime observer is restricted to `prooflayer_test`.
- The VM baseline is preserved as `day-10-windows-splunk-proof`.

See [Day 7 completion](docs/day-07-completion-report.md) and the
[lab runbook](docs/operations/lab-reproduction-runbook.md).

## Foundations

- [Shared data model](docs/architecture/shared-data-model.md)
- [Version 1 JSON contracts](schemas/v1/README.md)
- [Isolated Windows lab](deployments/lab/README.md)
- [Ideal customer profile](docs/strategy/ideal-customer-profile-v0.1.md)
- [First outreach drafts](docs/customer-discovery/day-03-outreach-drafts.md)
