# Technical Demo Script v0.1

**Audience:** SOC manager or detection engineer  
**Length:** Five minutes  
**Status:** Internal rehearsal draft; do not send under ADR-0002

## Opening

“ProofLayer proves whether one safe endpoint signal reaches Splunk with the
fields needed for detection, and identifies the first stage that fails.”

## Flow

1. Show the fixed `windows-process-marker@0.1.0` scenario and state that it
   accepts no command or payload.
2. Start one run and point to its unique `PL-` correlation ID.
3. Show execution PASS and cleanup PASS from the Runner audit.
4. Show the exact Sysmon Event ID 1 evidence without opening the raw event.
5. Show the exact Splunk match, ingestion latency, and field-presence result.
6. Switch to a synthetic missing-field result and explain that downstream
   detection and alert stages become `not_tested`, not false PASS results.

## Evidence language

- Say: “This stage passed because the required evidence was observed.”
- Say: “Detection and alert are not tested yet.”
- Do not say: “Your environment is secure.”
- Do not describe technical lab runs as customer traction.

## Closing question for the future MVP gate

“When a parser, forwarding route, or detection rule changes in your environment,
how do you prove today that the complete path still works?”
