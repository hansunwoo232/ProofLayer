# Day 3 — First Five Outreach Drafts

These drafts are personalized at the company and role level. Before sending,
replace `[Name]` with a verified current contact and choose an authorized
professional channel.

No message has been sent. Per ADR-0002, outreach is deferred until the first
repeatable Windows-to-Splunk MVP demonstration is ready.

## 1. Barikat — Managed Security Services Director

**Subject:** A short research call on detection-pipeline reliability

Hi [Name],

I am building a narrowly scoped detection reliability product for SOC and MSSP
teams. It safely validates whether a test behavior travels from Windows
telemetry through Splunk parsing, detection, and alert creation, and identifies
the exact stage where the chain breaks.

Barikat's managed security perspective would be especially valuable because the
same content and parser change can affect multiple customer environments. I am
not looking to sell a finished product; I would like to understand how your team
currently validates detection health and what a safe design-partner pilot would
require.

Would you be open to a 25-minute research call next week?

Best,  
Ahmet

## 2. Cyberwise — SOC or MDR Director

**Subject:** How do mature SOC teams test detection regressions?

Hi [Name],

I am researching a recurring SOC problem: a detection may remain enabled while
its telemetry source, field mapping, or alert path has silently stopped working.
The product we are prototyping runs harmless, signed test scenarios and checks
each step from endpoint event to SIEM detection.

Cyberwise operates at a level where repeatability and evidence matter, so I
would value your experience with parser changes, detection regression, and
customer-facing validation. This is a research conversation, not a sales demo.

Could we schedule a 25-minute call next week?

Best,  
Ahmet

## 3. Biznet Bilişim — SOC Services Manager

**Subject:** Research: validating the endpoint-to-SIEM detection path

Hi [Name],

I am working on a lightweight platform that answers one question: if a safe test
behavior occurs on a Windows endpoint today, will the expected Splunk detection
and alert actually appear?

The system is deliberately not an automated pentest or exploitation tool. It
uses allowlisted scenarios, correlation IDs, cleanup, and read-only SIEM
observation to isolate telemetry, ingestion, parsing, and rule failures.

I would appreciate 25 minutes to learn how Biznet's SOC services currently test
this path and what would make such a pilot safe and useful.

Best,  
Ahmet

## 4. ADEO Cyber Security — MDR or SOC Manager

**Subject:** Design research for continuous detection validation

Hi [Name],

We are designing a focused detection reliability product for teams that already
have EDR and SIEM coverage but cannot continuously prove that the complete
pipeline still works.

The first MVP is intentionally small: Windows, Sysmon/Event Log, Splunk, and a
few harmless test scenarios. The output is stage-level evidence—generated,
collected, ingested, parsed, detected, and alerted—plus a likely root cause.

Given ADEO's MDR perspective, I would value your feedback on operational fit,
customer approval, and the minimum evidence an MSSP would need. Would you be
available for a short research call next week?

Best,  
Ahmet

## 5. DnDx Cyber Security — SOC/MDR Lead

**Subject:** A lightweight “uptime monitor” for detection pipelines

Hi [Name],

I am testing a simple product thesis: SOC teams need an uptime monitor for
detections, not another vulnerability scanner. A safe scenario produces a
unique marker on a Windows test host; the platform then proves whether endpoint
telemetry, Splunk parsing, the rule, and the alert path worked.

For an MDR provider, the key question is whether this can become repeatable
evidence across customer environments without adding risky remote-execution
capability. DnDx's feedback would help us validate that assumption early.

Would you be open to a 25-minute problem interview next week?

Best,  
Ahmet
