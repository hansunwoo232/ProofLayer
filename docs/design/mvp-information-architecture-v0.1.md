# MVP Information Architecture v0.1

## Product frame

ProofLayer is a Detection Reliability workspace. The primary object is a Test
Run: one approved scenario executed on one host and observed through the
endpoint-to-alert pipeline.

## Primary navigation

| Area | Primary question | MVP surface |
|---|---|---|
| Overview | Is the detection pipeline healthy? | Health summary and recent failures |
| Test runs | What happened in each validation? | Run list and result detail |
| Scenarios | Which approved behaviors can run? | Scenario catalog and safety metadata |
| Hosts | Where can validations run? | Registered host and Runner health |
| Schedules | When should tests run? | One-time plans |
| Reports | What evidence can be shared? | JSON and HTML result exports |
| Audit log | Who requested or changed what? | Immutable operator activity timeline |
| Settings | How is this workspace configured? | Workspace, users, integrations, security |

Days 31–35 implement authentication, the existing Test Run detail, test
creation, host health, one-time schedules, and filtered history. Overview,
reports, audit, and settings remain planned.

## Test Run detail hierarchy

1. Overall outcome, scenario, host, completion time, and correlation ID.
2. Ordered pipeline: execution, endpoint telemetry, SIEM ingestion, field
   validation, detection, alert, and cleanup.
3. Root cause with bounded remediation guidance.
4. Expected-versus-observed evidence without raw customer event bodies.
5. Technical metadata and audit identifiers.

## Core entity relationships

```text
Workspace
  ├── Users
  ├── Hosts ── Runner
  ├── Scenarios
  ├── Schedules
  └── Test Runs
        ├── Stage Results
        ├── Evidence Summary
        └── Audit Events
```

Every user, host, scenario, schedule, and run belongs to exactly one workspace
in the MVP contract. Cross-workspace access is never inferred from a browser
request.

## Roles

The local MVP exposes one `admin` role. The information architecture reserves
the following future role boundaries without implementing them:

- administrator: workspace security, users, integrations, and all test actions;
- operator: run approved scenarios and inspect evidence; and
- viewer: read results and reports without mutation rights.

## Interaction rules

- unauthenticated visitors enter through Sign in;
- successful sign-in returns to the workspace and Test Run result;
- Sign out revokes the current session and returns to Sign in;
- destructive or active validation actions remain explicit and attributable;
- raw credentials, raw endpoint events, and customer data are excluded from
  list and summary surfaces; and
- an unavailable feature is not shown as active navigation during the MVP.

## Route direction

| Route | Purpose | Day 31 state |
|---|---|---|
| `/login.html` | Local workspace sign-in | Implemented |
| `/` | Authenticated workspace entry | Redirects to current run detail |
| `/result-screen-wireframe.html` | Test Run result detail | Implemented |
| `/history.html` | Filtered Test Run list | Implemented |
| `/test-new.html` | Host and scenario selection | Implemented |
| `/hosts.html` | Host inventory and Runner health | Implemented |
| `/schedules.html` | One-time test schedules | Implemented |
| `/reports` | Evidence exports | Planned |
| `/audit` | Operator audit log | Planned |
| `/settings` | Workspace configuration | Planned |
