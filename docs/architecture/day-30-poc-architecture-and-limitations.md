# Day 30 PoC Architecture and Known Limitations

**Status:** Implemented; live Windows acceptance evidence pending

**Date:** August 18, 2026

## One-action path

1. The operator clicks **Run Test** for the fixed Windows Process Marker.
2. The Control Plane creates a two-minute, host-bound Ed25519-signed job.
3. The outbound-only Windows worker leases the job over the isolated TLS
   guest forward, verifies its signature, expiry, identity binding, nonce, and
   empty parameter set, then acknowledges it.
4. The fixed executor emits one correlation marker and performs mandatory
   cleanup verification.
5. The local Sysmon observer proves Event ID 1 without returning a raw event.
6. The HEC exporter submits only event metadata and fixed synthetic canary
   values to `prooflayer_test`.
7. The read-only Splunk Observer verifies exact ingestion, the four required
   field names, and the built-in detection result.
8. The worker publishes ordered terminal stages. Alert remains `not_tested`
   because no SOAR or alert-delivery integration exists in the PoC.

## Trust boundaries

- Browser traffic binds to `127.0.0.1:8787`.
- Runner transport binds to `127.0.0.1:8788` with TLS and is reachable from the
  guest only as `10.0.2.100:8788`.
- HEC and Splunk management remain host-loopback services exposed to the guest
  only by fixed QEMU forwards.
- The Runner accepts no command, executable, argument, path, payload, or custom
  scenario parameter from the Control Plane.
- Lab secrets, TLS material, generated config, VM state, and ISO media are
  ignored local artifacts.

## Known limitations

- Queue, run history, and lifecycle state are in memory and disappear when the
  Control Plane restarts.
- The lab uses a bearer token plus a short-lived self-signed TLS certificate;
  production requires mutual TLS, durable registration, rotation, and
  revocation.
- Nonce replay state is in memory and bounded to one process lifetime.
- The Day 30 executable processes one job and exits; the persistent service,
  heartbeat, bounded backoff, cancellation, and offline buffering remain open.
- HEC receives safe synthetic field values after local Sysmon proof. This
  validates the transport/parser/detection path without exporting the original
  Windows command line or username.
- Only Windows Process Marker is accepted by the Day 30 worker. Registry and
  Scheduled Task canaries remain independently proven lab harnesses.
- Alert delivery is intentionally `not_tested`.
- The self-signed certificate exception is compiled only into the fixed Day 30
  lab harness and is bound to three non-routable guest-forward URLs.

These limits are acceptable for the technical PoC and are not acceptable for
a customer or production deployment.
