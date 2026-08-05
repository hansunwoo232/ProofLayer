# Day 18 Completion Report

**Date:** August 5, 2026  
**Status:** Complete in the isolated Windows lab.

## Completed

- Implemented exact-correlation Sysmon Event ID 1 observation in the Go Runner.
- Added the frozen 15-second deadline, 500 ms polling interval, 30-attempt
  ceiling, 60-second query lookback, 50-event query cap, and 512 KiB XML cap.
- Kept event bodies and endpoint field values inside the Windows boundary.
- Added stable handling for invalid IDs, source failures, deadline expiry, and
  bounded event-not-found results.
- Reproduced event-not-found behavior in automated tests and confirmed no query
  occurs for an invalid correlation ID.
- Verified the Day 17 correlation ID against the real Sysmon channel.

## Validated endpoint evidence

| Field | Result |
|---|---|
| Correlation ID | `PL-0FB5965F8BD19BFEFAD8F30F5944B8E9` |
| Provider | `Microsoft-Windows-Sysmon` |
| Event ID | `1` |
| Record ID | `2762` |
| Event time UTC | `2026-08-05T19:24:37.3685402Z` |
| Exact-correlation lookup | PASS |

Ignored lab screenshot:
`work/prooflayer-lab/evidence/day-18-sysmon-observation-pass.png`

## Next gate

Connect the Runner-side Observer to Splunk using the existing search-only
identity and prove authenticated health, invalid-credential, and unreachable
endpoint behavior.

## Deferred under ADR-0002

Detection-engineer review of the result language remains deferred until the
visible MVP gate.
