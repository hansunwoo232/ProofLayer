# Day 33 Completion Report

## Outcome

The Hosts surface shows workspace-bound host status, authenticated runner
last-seen time, runner version, and authorized scenario count.

## Acceptance evidence

- A host with no recent authenticated activity is shown as offline.
- An authenticated runner request records last-seen time and version.
- Status becomes offline after the two-minute health window.
- Health does not grant access; host/scenario authorization remains explicit.
- The Runner sends a bounded version header on Control Plane requests.

## Access rule

Only the configured runner identity may update the configured host. Only the
fixed catalog IDs and exact versions authorized for that host may be queued.
