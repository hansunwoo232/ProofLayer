# Day 34 Completion Report

## Outcome

ProofLayer supports one-time scheduled validation tests in Europe/Istanbul or
UTC through a four-field form: host, scenario, local time, and time zone.

## Acceptance evidence

- Europe/Istanbul local time is converted to the correct UTC instant.
- The original local value and zone are retained for display and audit.
- Past or insufficient-lead plans return `SCHEDULE_TIME_PASSED`.
- Same-host plans less than 60 seconds apart return `SCHEDULE_CONFLICT`.
- Plans more than five minutes late become `missed`.
- Due plans enter the existing idempotent queue and become `queued`.

The UI provides direct, non-technical guidance for past and conflicting plans.
