# Technical Demo Script v0.1

**Audience:** SOC manager or detection engineer  
**Length:** Five minutes  
**Status:** Internal rehearsal draft; do not send under ADR-0002

## Opening

“ProofLayer proves whether one safe endpoint signal reaches Splunk with the
fields needed for detection, and identifies the first stage that fails.”

## Flow

1. **0:00–0:45 — Scope.** Show the fixed
   `windows-process-marker@0.1.0` scenario and state that it accepts no command,
   argument, parameter, or payload.
2. **0:45–1:15 — Start.** Select Run Test and point to its unique `PL-`
   correlation ID and duplicate-safe queue receipt.
3. **1:15–2:15 — Live state.** Show queued, leased, acknowledged, and running
   transitions. Explain that a delayed event remains IN PROGRESS within a
   bounded window rather than becoming a premature FAIL.
4. **2:15–3:15 — Evidence.** Show execution PASS, Sysmon Event ID 1, exact
   Splunk correlation, and ingestion latency without opening the raw event.
5. **3:15–4:15 — Root cause.** Show the missing-field case. Explain that
   downstream detection and alert become NOT TESTED, not false PASS results.
6. **4:15–5:00 — Safety and close.** Show cleanup as an independent stage,
   state that the browser cannot send lifecycle updates, and ask the closing
   question below.

## Current rehearsal boundary

The five-minute narrative and UI states are prepared for internal rehearsal.
Do not present the local queue as a complete live demo until authenticated
outbound Runner transport supplies the lifecycle updates. The current UI never
fabricates PASS evidence.

## Evidence language

- Say: “This stage passed because the required evidence was observed.”
- Say: “Detection and alert are not tested yet.”
- Do not say: “Your environment is secure.”
- Do not describe technical lab runs as customer traction.

## Closing question for the future MVP gate

“When a parser, forwarding route, or detection rule changes in your environment,
how do you prove today that the complete path still works?”
