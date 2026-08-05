# Day 8 Rollover

## Completed or absorbed by earlier work

- The lab installation and checkpoint workflow is documented and repeatable.
- The event model has versioned JSON examples and dependency-free invariant
  checks.
- The Runner negative capability boundary is explicit.
- Risk and assumption tracking is active.

## Deferred under ADR-0002

- Two additional discovery interviews
- SIEM migration interview targeting
- Interview-derived user-need prioritization
- Outreach conversion-rate measurement

These are deferred, not completed, and must not be reported as traction.

## Technical backlog status on August 5, 2026

1. Complete the first Windows → Sysmon → Splunk proof: **complete**.
2. Repeat the proof three times and measure latency: **complete**.
3. Add least-privilege Splunk observer access: **complete**.
4. Define the standalone validator interface before Go Runner implementation:
   **design complete; implementation remains in the product backlog**.
5. Keep detection and alert validation out of scope until ingestion is stable:
   **still in force**.
