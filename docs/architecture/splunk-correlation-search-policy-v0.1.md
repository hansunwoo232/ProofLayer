# Splunk Correlation Search Policy v0.1

**Status:** Frozen for the technical MVP  
**Date:** August 5, 2026

## Query boundary

- Fixed index: `prooflayer_test`
- Fixed source: `prooflayer:windows-lab`
- Exact canonical correlation ID only
- Event ID: `1`
- Maximum search window: 24 hours
- Maximum result rows: two, so duplicate evidence fails closed
- Returned fields: correlation ID, provider, Event ID, record ID, endpoint event
  time, and ingestion latency

The query never returns `_raw`, command-line values, usernames, hostnames, or
arbitrary event fields. All user-controlled search fragments are prohibited;
only a validated canonical correlation ID and numeric UTC epoch bounds enter
the fixed SPL template.

## Polling boundary

| Control | Allowed range | Default |
|---|---:|---:|
| Overall timeout | >0 to 60 seconds | 60 seconds |
| Poll interval | 250 ms to 5 seconds | 2 seconds |
| Maximum attempts | 1 to 120 | 30 |

Only a valid zero-result response is retried. Authorization, connection,
parsing, duplicate-result, or scope errors stop immediately. A late-event test
returns no event twice and the exact event on the third bounded attempt.

## Result validation

The connector independently checks that the returned correlation ID equals the
requested ID, the provider is `Microsoft-Windows-Sysmon`, Event ID is `1`, all
numeric fields parse, and the event timestamp is RFC 3339. Zero rows is
`not_found`; two rows is `ambiguous`; neither can produce PASS.
