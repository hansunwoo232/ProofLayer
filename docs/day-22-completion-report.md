# Day 22 Completion Report

**Date:** August 5, 2026  
**Status:** Complete within the approved technical workstream.

## Completed

- Added the detection-result control after the exact Splunk observation layer.
- Implemented fixed built-in inline SPL and fixed Saved Search query plans.
- Kept detection ID, Saved Search name, query structure, index, source, result
  fields, and ambiguity ceiling inside compiled code.
- Added stable detection present and absent results.
- Added fail-closed tests for duplicate results, invalid plans, different Saved
  Search names, and scope relaxation attempts.
- Added minimum-metadata present and absent JSON examples.
- Completed the Splunk detection test-data preparation guide.

## Test result

```text
Inline detection present:          PASS
Detection absent result:           PASS
Duplicate detection fail-closed:   PASS
Saved Search abstraction:          PASS
Arbitrary plan rejection:          PASS
Raw event return surface:          NONE
```

## Current evidence limit

The inline and Saved Search plans are covered by deterministic connector tests.
The existing live Splunk evidence proves the underlying exact event; a Saved
Search has not been provisioned in the founding lab. No live Saved Search claim
is made.

## Deferred under ADR-0002

The detection result has not been shown to a prospect. Customer-facing review
remains gated by the visible MVP and cleanup demonstration.
