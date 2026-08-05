# Day 21 Completion Report

**Date:** August 5, 2026  
**Status:** Complete within the approved technical workstream.

## Completed

- Added boolean-only field-presence evidence to the fixed Splunk exact-ID
  query for the four scenario-required fields.
- Implemented the field validator with deterministic present/missing lists,
  counts, and PASS/FAIL status.
- Added the explicit missing `process.command_line` parser-failure fixture.
- Added tests for all-present, missing-command-line, absent evidence, unknown
  evidence keys, and serialized-result data minimization.
- Verified that command lines, usernames, `_raw`, and arbitrary endpoint values
  cannot appear in the validation result type.
- Rebuilt the Runner for Windows ARM64 after the validator integration.
- Preserved the stopped VM, UEFI, and TPM state in checkpoint
  `day-21-runner-observer-proof`.

## Evidence basis

The live Day 12 three-run series already proved all four required fields present
in Splunk. Day 21 adds the product implementation and negative parser-failure
path without modifying the preserved live evidence.

## Deferred under ADR-0002

The missing-field output has not yet been shown to a prospect or user. It is
ready for the visible MVP demonstration gate.
