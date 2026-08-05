# Splunk Detection Test Data Guide

**Scope:** Isolated ProofLayer lab only  
**Security:** Never use production credentials or customer data

## Prerequisites

- Splunk lab is healthy and bound to loopback.
- `prooflayer_test` exists.
- HEC ingestion identity and search-only `prooflayer_observer` are provisioned.
- Windows Sysmon channel is enabled.
- The fixed process-marker scenario is approved.

## Produce one test event

1. Start the Splunk lab and wait for health.
2. Start the Windows lab in SIEM network mode.
3. Run the fixed `prooflayer-runner-lab` binary from read-only media.
4. Record its canonical correlation ID.
5. Use the existing lab HEC bridge to submit the matching normalized Sysmon
   evidence into `prooflayer_test`.
6. Verify the exact ID with `verify-windows-event-by-id.sh`.

The committed repository contains no HEC token or observer password. Ignored
local credentials must be injected through the lab environment and must never
be pasted into screenshots, commands, reports, or Git history.

## Inline detection rule

The built-in connector plan implements this fixed logic:

```spl
search index=prooflayer_test source="prooflayer:windows-lab" earliest=<UTC_EPOCH> latest=<UTC_EPOCH> "<CANONICAL_ID>"
| spath
| eval correlation_id=mvindex(correlation_id,0), event_id=mvindex(event_id,0)
| where correlation_id="<CANONICAL_ID>" AND event_id=1
| eval detection_id="prooflayer.windows_process_marker"
| head 2
| table correlation_id,detection_id
```

Only the validated ID and numeric epoch bounds vary.

## Optional Saved Search

Create a Saved Search named exactly `ProofLayer Windows Process Marker` with the
same logic and these validated substitution tokens:

- `$correlation_id$`
- `$earliest_epoch$`
- `$latest_epoch$`

Do not grant the runtime Observer permission to create, edit, schedule, or
delete Saved Searches. Provisioning remains an administrator-only operation;
runtime access is search-only.

## Acceptance cases

1. **Present:** one exact detection row produces `detected=true`.
2. **Absent:** zero rows produces `detected=false` without a connector error.
3. **Ambiguous:** two rows fails closed.
4. **Mismatch:** a different correlation or detection ID is rejected.
5. **Scope:** `_internal` remains unavailable to the Observer.

Delete no lab volume after testing. Stop Splunk with Compose so the evidence
remains available for the next regression run.
