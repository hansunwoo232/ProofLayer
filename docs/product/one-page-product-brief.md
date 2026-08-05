# ProofLayer — one-page product brief

> **Code name:** ProofLayer. Until brand clearance is complete, external
> communication should use the category name “Detection Reliability Platform.”

## Problem

Having EDR, Sysmon, forwarding, SIEM rules, and a SOC process does not prove an
attack behavior will be detected. Endpoint telemetry, ingestion, parsing,
detection, or alert delivery can fail silently. Teams often discover the gap
through manual checks or only after an incident.

## Initial customer

The initial user manages 250–5,000 endpoints, uses Windows, Sysmon or Windows
Event Log, and Splunk, has a 5–30 person SOC or detection team, and experiences
parser, log-source, or SIEM changes. Buyers include SOC Managers, Detection
Engineering Leads, SIEM Engineers, and MSSP service leaders.

## Solution

ProofLayer runs a predefined harmless behavior on an authorized Windows host
and follows one correlation ID through the pipeline:

```text
Safe behavior → Endpoint telemetry → Splunk ingestion
              → Required fields → Detection → Alert
```

Each stage receives an explicit result. The product explains a verifiable root
cause, such as a missing `process.command_line` field, instead of returning only
a generic failure.

## Differentiation

ProofLayer is not a vulnerability scanner, automated pentest tool, or broad BAS
platform. It validates whether the detection pipeline works today, avoids real
exploitation and credential dumping, and reruns the same test after a fix to
produce regression evidence.

## Initial MVP

- Windows Runner and Sysmon/Windows Event Log observation
- Splunk integration and six safe scenarios
- Correlation IDs and stage-level results
- Test history, scheduling, and reporting
- Signed scenarios, allowlists, cleanup, and a kill switch
- Synthetic public product demo

## Security principle

The Runner cannot execute arbitrary commands. It accepts only signed,
allowlisted, time-bounded scenarios with required cleanup on registered hosts.
Only test metadata leaves the customer environment.

## Outcome sold

> Know which detections are broken before attackers do.

We seek an internal SOC and an MSSP willing to evaluate three to six safe
scenarios in a controlled one-week design-partner pilot.
