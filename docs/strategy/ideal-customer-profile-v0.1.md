# Ideal Customer Profile Hypothesis v0.1

**Date:** July 29, 2026  
**Status:** Hypothesis to validate through 20–30 interviews  
**Primary market:** Turkey first, English-language global product

## Core hypothesis

The first paying ProofLayer customer is most likely a mid-market organization or
MSSP that already spends materially on endpoint telemetry and SIEM, has enough
detection content to experience regression risk, but lacks a dedicated platform
or team for continuous detection-pipeline validation.

## Primary ICP — internal SOC

### Firmographics

- 250–5,000 managed endpoints
- 5–30 people across SOC, SIEM, detection, and incident response
- Regulated or high-availability digital business
- Security tooling budget exists; procurement is not purely project-based
- One or more recent SIEM, parser, EDR, or log-pipeline changes

### Technographics

- Windows-heavy endpoint estate
- Splunk is the initial SIEM
- Sysmon, Windows Event Log, EDR telemetry, or a combination
- More than 50 production detection rules
- Detection changes are versioned or at least centrally managed
- API access can be provided to a controlled test environment

### Strong trigger events

- SIEM migration or index/sourcetype redesign
- Parser or data model change
- EDR migration or policy change
- Recent silent log-source outage
- Audit finding related to control effectiveness
- SOC service-level review after a missed detection
- New detection engineering lead tasked with measuring coverage

### Primary users and buyers

| Role | Primary concern |
|---|---|
| Detection Engineer | Regression, missing fields, repeatable test evidence |
| SIEM Engineer | Ingestion, parsing, latency, source health |
| SOC Manager | Detection health, ownership, audit and executive evidence |
| CISO / Security Director | Control effectiveness and risk reduction |

## Secondary ICP — MSSP

### Why it may be the best channel

- One deployment can expose the product to multiple customer environments.
- The MSSP needs repeatable evidence that a managed detection service works.
- Standardized tests can become a differentiated service package.
- Multi-customer parser and content changes create recurring regression risk.

### Additional requirements

- Strict tenant separation
- Per-customer approval and maintenance windows
- White-label or co-branded reporting
- Role separation between service delivery and customer operators
- Commercial model based on tenants or environments, not endpoint count

Multi-tenancy remains outside the 90-day MVP; interviews should validate demand
without pulling the feature into the first build.

## Initial vertical priority

1. MSSPs and managed SOC providers
2. Fintech and payment companies
3. B2B SaaS and digital marketplaces
4. Telecommunications
5. Mid-sized banks

Critical infrastructure and defense are strategically relevant but are not the
first design-partner target because procurement, security review, and
deployment timelines are longer.

## Disqualifiers

- No SIEM or centralized detection process
- Fewer than 20–30 meaningful detection rules
- Only compliance checkbox interest; no operational owner
- No controlled Windows test host
- Requires active exploitation, credential dumping, or ransomware simulation
- Requires cloud, Linux, OT, Sentinel, or Elastic in the first pilot
- Cannot provide a safe test window or read-only SIEM API access
- Procurement cannot start within six months

## Pilot-ready profile

A prospect is pilot-ready when all of the following are true:

- A technical owner is named.
- A controlled Windows host is available.
- Splunk read-only access can be approved.
- Three safe scenarios are accepted.
- A one-week test window is available.
- Success is defined as a verified gap, saved engineering time, or repeatable
  evidence—not a guaranteed finding.

## ICP score

Score each account from 0–2:

| Dimension | 0 | 1 | 2 |
|---|---|---|---|
| Detection maturity | Ad hoc | Managed rules | Dedicated detection function |
| Stack fit | No Splunk/Windows | Partial fit | Windows + Splunk |
| Pain signal | None | Suspected | Recent verified failure |
| Trigger | None | This year | Active migration/change |
| Pilot access | Unclear | Controlled host | Owner, host, API, date |
| Buying path | Unknown | Identified | Budget owner engaged |

- **0–4:** Do not prioritize
- **5–7:** Discovery
- **8–10:** Demo target
- **11–12:** Design-partner target

## Falsification criteria

Revise the ICP if, after 15 interviews:

- Fewer than five prospects report a real pipeline failure in the last year.
- Splunk users already solve the problem reliably with internal automation.
- Pilot security requirements make a lightweight deployment impossible.
- MSSPs require multi-tenancy before they can evaluate any technical value.
- The buyer and operational owner consistently sit in different budgets with no
  joint decision path.

