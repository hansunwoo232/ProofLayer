# Phase 3 MVP Backlog

**Scope:** Days 31–45

**Prioritization date:** August 18, 2026

**Input:** Day 30 technical gate

## P0 — required for the Day 45 gate

1. Persist users, workspaces, hosts, scenarios, runs, stages, and audit events
   in PostgreSQL with migrations and strict tenant keys.
2. Add local operator authentication, secure password hashing, session expiry,
   CSRF continuity, and logout.
3. Replace the one-shot lab executable with a bounded Windows service loop,
   heartbeat, version reporting, retry backoff, cancellation, and kill switch.
4. Persist job leases, idempotency records, signing key references, and nonce
   consumption so restarts fail closed without losing run state.
5. Implement host registration and per-host authorization with rotation and
   revocation-ready credentials.
6. Complete run history, run detail, host list, scenario selection, and one-time
   scheduling surfaces against persisted APIs.
7. Produce versioned redacted JSON and HTML reports from stored evidence.
8. Run process-marker, Registry Run Key, and Scheduled Task scenarios three
   times each and prove deterministic cleanup and schema compatibility.

## P1 — required before external design-partner access

1. Replace the isolated self-signed exception with CA validation and mutual TLS.
2. Add admin, operator, and viewer roles with authorization tests.
3. Add tamper-evident audit chaining and a documented retention policy.
4. Add degraded latency thresholds, clock-skew rejection, and clear root-cause
   classes for telemetry, ingestion, parser, detection, and timeout failures.
5. Create a clean-lab installer and complete a second-person installation test.

## P2 — after Day 45 evidence

1. Add email or Slack notifications.
2. Add Elastic or Sentinel only after the Splunk contract is stable.
3. Add multi-tenant MSSP structure only after one single-tenant pilot is
   repeatable.

## Explicitly deferred

- Customer outreach and demo follow-up remain deferred under ADR-0002 until the
  technical MVP has repeatable evidence.
- AI remediation, arbitrary commands, exploitation, cloud, Kubernetes, OT,
  SOAR, and attack-path features remain outside MVP scope.
