# Day 20 Completion Report

**Date:** August 5, 2026  
**Status:** Complete with live local Splunk validation.

## Completed

- Added exact canonical correlation search to the Go Splunk connector.
- Enforced the fixed index, source, Event ID, SPL template, numeric UTC bounds,
  24-hour maximum window, and two-row ambiguity ceiling.
- Added configurable but bounded polling timeout, interval, and attempt count.
- Added late-event, exhausted-not-found, empty, duplicate, result-mismatch, and
  invalid-window tests.
- Ensured search output contains metadata only and never raw events or endpoint
  field values.
- Built the exact-search utility for Windows ARM64.
- Prepared the first internal five-minute technical demo script.

## Live validation

| Field | Result |
|---|---|
| Correlation ID | `PL-1188F1C905964BB7863D97CE8E6BBEB8` |
| Attempts | 1 |
| Provider | `Microsoft-Windows-Sysmon` |
| Event ID | 1 |
| Record ID | 2447 |
| Endpoint event time | `2026-08-05T18:41:41.323Z` |
| Ingestion latency | 2677 ms |
| Exact result | PASS |

## Deferred under ADR-0002

The technical demo script is for internal rehearsal only. It has not been sent
to or tested with a prospect.
