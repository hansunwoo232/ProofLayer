# MVP Workspace Surfaces v0.1

## Purpose

This slice connects ProofLayer's local authenticated dashboard to one bounded,
in-memory workspace model. It covers test creation, host health, one-time
scheduling, and test history without expanding the Day 30 execution boundary.

## Read model

- `GET /v1/scenarios` returns the fixed scenario catalog with version, risk
  level, expected effects, and cleanup requirement.
- `GET /v1/hosts` returns the workspace-bound host, runner version, last-seen
  time, online/offline status, and authorized scenario IDs.
- `GET /v1/schedules` returns one-time plans newest first.
- `GET /v1/test-runs` returns newest-first history with optional host, scenario,
  UTC date range, page, and page-size filters.

Every browser read requires the authenticated local session and its CSRF token.

## Write model

`POST /v1/test-jobs` requires the configured environment, host, scenario, and
exact scenario version. Host mismatch returns `HOST_ACCESS_DENIED`; a catalog or
version mismatch returns `SCENARIO_INVALID`.

`POST /v1/schedules` accepts an approved host/scenario pair, a local timestamp,
and either `Europe/Istanbul` or `UTC`. The service converts the timestamp to UTC
while preserving the submitted local value and zone. It rejects past times,
plans more than 30 days ahead, and plans for the same host less than 60 seconds
apart.

The scheduler dispatches due plans through the existing idempotent queue. A plan
more than five minutes late becomes `missed` instead of executing unexpectedly.

## Host access and health

The configured runner identity is the only identity allowed to update last-seen
and version. A host is online when an authenticated runner request was observed
within two minutes. Health is informational and never changes authorization.

The runner sends `X-ProofLayer-Runner-Version` on every authenticated request.
The Control Plane stores only the bounded version string and timestamp.

## History indexing and retention

The in-memory queue keeps insertion order plus host and scenario ID indexes.
Filtered reads use the narrowest available index and return a maximum of 50
items per page. Deleted lifecycle records are skipped during reads. Durable
history, index compaction, and cross-process persistence remain post-MVP work.

## Security properties

- No arbitrary commands, executable arguments, or raw event bodies enter these APIs.
- Scenario selection is catalog- and version-bound.
- Browser mutations require exact loopback Origin, authenticated session, and CSRF.
- Dashboard code uses DOM text nodes and no browser storage.
- Schedules cannot bypass the existing queue, signature, or runner identity checks.
