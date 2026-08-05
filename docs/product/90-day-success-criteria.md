# 90-day success criteria

**Period:** July 27–October 24, 2026  
**Code name:** ProofLayer  
**Principle:** Success requires a working product, user evidence, and commercial
signal—not software delivery alone.

## North star

After a controlled Windows test starts, ProofLayer must prove which stage
worked or failed before the configured observation window ends:

`Execution → Endpoint telemetry → SIEM ingestion → Field validation → Detection
→ Alert → Cleanup`

## Day 90 minimum threshold

### Product

- The Windows Runner executes only approved scenarios.
- At least four, with a target of six, safe scenarios run end to end.
- Sysmon or Windows Event Log and one Splunk environment are supported.
- A correlation ID joins execution, event, and detection evidence.
- Stages use `passed`, `failed`, `degraded`, or `not_tested` semantics.
- Missing telemetry, missing fields, detection failure, and timeout are distinct.
- Test history, hosts, scenarios, and pipeline details are visible.
- JSON and printable HTML reports are available.

### Security and quality

- The Runner cannot execute arbitrary commands.
- Scenarios have integrity or signature checks, allowlists, timeouts, and a kill
  switch.
- Every state-changing scenario performs and records automatic cleanup.
- No real customer data or credentials leave the environment.
- A clean lab can be installed from documentation without developer help.
- The same scenario produces consistent outcomes in three consecutive runs.

### Customer evidence

- Minimum 20, target 30+ problem interviews.
- Minimum five, target 10+ live product demos.
- At least one, target three design partners.
- At least one written pilot intent or LOI; strong target: one paid pilot.
- At least one real telemetry, parser, or detection gap found and retested.

### Go to market

- A public synthetic demo works without a real Runner or SIEM connection.
- Marketing pages explain the product, workflow, security, and demo request.
- A two-minute product video and one-page brief exist.
- A 10–12 slide deck, 18-month model, and investor data room are ready.
- Minimum three, target 10 investor conversations begin.

## Target scorecard

| Metric | Minimum | Target |
|---|---:|---:|
| Safe scenarios | 4 | 6 |
| SIEM integrations | 1 | 1 |
| Problem interviews | 20 | 30+ |
| Live demos | 5 | 10+ |
| Design partners | 1 | 3 |
| Written pilot intent | 1 | 2–3 |
| Paid pilots | 0 | 1 |
| Public demo | Working | Generates demand |
| Waitlist | 20 | 50+ |
| Investor conversations | 3 | 10 |

## Decision gates

- **Day 14:** 10 interviews, five strong signals, two demo/pilot candidates, and
  one Windows-to-Splunk event.
- **Day 30:** One action shows Runner execution, Windows event, Splunk, and
  detection outcome.
- **Day 45:** Three repeatable scenarios, reliable cleanup, and visible results.
- **Day 60:** Another person can install the product in a clean lab from docs.
- **Day 72:** Pilot finding, user quote, and commercial intent.
- **Day 82:** Public synthetic demo and demand-capture flow are live.

## Pivot triggers

- Fewer than five strong signals by Day 14: narrow the ICP or problem.
- No end-to-end chain by Day 30: stop new scenarios and UI work.
- No safe independent installation by Day 60: do not approach production pilots.
- No pilot or payment signal by Day 72: extend discovery before investor outreach.
