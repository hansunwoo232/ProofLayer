# Endpoint Observation Policy v0.1

**Status:** Frozen for `windows-process-marker@0.1.0`  
**Date:** August 5, 2026

## Expected event

- Channel: `Microsoft-Windows-Sysmon/Operational`
- Provider: `Microsoft-Windows-Sysmon`
- Event ID: `1`
- Correlation: exact canonical ID in an event data value
- Earliest accepted event: execution start minus two seconds for clock and
  callback ordering tolerance

## Polling boundary

| Control | Approved value |
|---|---:|
| Overall deadline | 15 seconds |
| Poll interval | 500 ms |
| Maximum attempts | 30 |
| Windows query lookback | 60 seconds maximum |
| Windows query minimum | 10 seconds |
| Events per query | 50 maximum |
| Local XML buffer | 512 KiB maximum |

The policy may be made stricter but cannot be relaxed by a scenario. A valid
empty query is retried. Access denial, malformed XML, evidence overflow, or a
source failure stops immediately. Deadline or exhausted attempts produces the
stable `endpoint_event_not_found` outcome.

The query window is intentionally wider than the evidence acceptance window.
This accommodates delayed channel publication without accepting an event older
than execution start minus two seconds.

## Metadata boundary

The Windows source inspects event data locally to locate the exact ID. The
Observer returns provider, Event ID, record ID, timestamps, and attempt count.
It never returns the command line, username, or raw XML.

## Negative proof

Automated tests query a source that never returns the target and confirm exactly
three bounded attempts in the shortened test policy, followed by
`ErrEventNotFound`. Invalid correlation IDs are rejected before any source
query.
