# MVP scope

## Product definition

ProofLayer tests the path from a safe behavior on a Windows endpoint to a
Splunk detection and alert, then identifies the failed stage and likely cause.

## In scope

- Windows Runner
- Sysmon and Windows Event Log validation
- Splunk REST API integration
- Unique correlation IDs
- Execution, telemetry, ingestion, field, detection, alert, and cleanup stages
- One-time and scheduled tests
- Test history and pipeline details
- Six target safe scenarios: PowerShell marker, Registry Run Key canary,
  Scheduled Task canary, local-user canary, Windows-service canary, and DNS
  canary query
- Automatic cleanup
- Signed, allowlisted scenario packages
- Admin, operator, and viewer roles
- Audit log
- JSON and printable HTML reports
- Docker Compose control-plane deployment
- A synthetic public demo that never connects to customer systems

## Out of scope

- Exploitation or vulnerability scanning
- Credential dumping, pass-the-hash, or privilege escalation
- Ransomware or file-encryption simulation
- Real data exfiltration
- Arbitrary command execution
- Linux Runner
- Elastic, Sentinel, EDR, or SOAR integrations
- Cloud, Kubernetes, Active Directory attack paths, or OT
- Automatic remediation or an LLM agent
- MSSP multi-tenancy, native mobile apps, SAML/OIDC, or enterprise provisioning

## Test acceptance criteria

1. Select a registered, in-scope host.
2. Validate the signed scenario.
3. Generate a unique correlation ID.
4. Execute the safe behavior within its timeout.
5. Find the expected endpoint telemetry.
6. Find the event and required fields in Splunk.
7. Check the detection and, where possible, alert result.
8. Run and verify cleanup.
9. Record every stage result, latency, and error reason in the audit log.
10. Do not send raw customer content to the central service.

## Scope-change rule

During the first 90 days, a new feature enters scope only if the same need is
observed in at least three customer interviews, the current decision gate stays
safe, and security and test acceptance criteria are written first.
