# Day 17 Completion Report

**Date:** August 5, 2026  
**Status:** Complete in the isolated Windows lab.

## Completed

- Implemented the fixed `builtin.emit_process_marker` Windows handler.
- Kept executable path, arguments, output handling, network behavior, and
  scenario version inside compiled code; the handler accepts only a validated
  canonical correlation ID.
- Applied the approved execution deadline and discarded process output.
- Added stable results for invalid input, unsupported platform, execution
  failure, deadline expiry, and audit failure.
- Added an isolated-lab harness that accepts no arguments and cannot select a
  command or scenario.
- Built and executed the Windows ARM64 lab binary from read-only ISO media.
- Recorded execution PASS, cleanup PASS, and a 58 ms execution latency.

## Validated run

| Field | Result |
|---|---|
| Scenario | `windows-process-marker@0.1.0` |
| Correlation ID | `PL-0FB5965F8BD19BFEFAD8F30F5944B8E9` |
| Execution | PASS |
| Execution latency | 58 ms |
| Cleanup | PASS |
| Local audit | `C:\ProgramData\ProofLayer\runner-audit.jsonl` |

Ignored lab screenshot:
`work/prooflayer-lab/evidence/day-17-runner-pass.png`

## Safety conclusion

The observed behavior matched the approved process-marker boundary. This result
does not yet prove endpoint observation; exact-ID Sysmon verification is the Day
18 gate.

## Deferred under ADR-0002

Customer review of result language remains deferred until the visible MVP gate.
