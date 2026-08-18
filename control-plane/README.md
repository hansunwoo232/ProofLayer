# Control Plane

The Control Plane owns Runner registration, scheduling, correlation, audit
events, and result APIs. ADR-0005 selects Go for the technical PoC.

## Day 29 local process

From the `control-plane` directory:

```bash
go run ./cmd/prooflayer-control-plane
```

Open `http://127.0.0.1:8787`. The process serves the result surface and accepts
one fixed, allowlisted `Run Test` request. It binds only to loopback.

The browser and lifecycle implementation provides:

- exact-origin and double-submit CSRF validation;
- strict, size-limited JSON parsing;
- server-side environment, host, scenario, and version binding;
- duplicate-safe idempotency handling;
- a bounded concurrency-safe queue;
- two-minute Ed25519-signed Test Jobs; and
- a one-time host-bound lease interface;
- fail-closed acknowledgement and ordered stage transitions; and
- a session-bound read-only status endpoint for bounded live UI polling.

The lifecycle accepts only the seven canonical stages. Detail text cannot be
submitted by a Runner; updates use a small stable `detail_code` allowlist. A
failed upstream stage can only move downstream stages to `not_tested`, while
cleanup remains mandatory before terminal completion.

This is a local-only process. Its queue and signing key are ephemeral, it has no
operator authentication or durable audit store, and it must not be exposed to
a LAN, customer environment, or public network. See the
[Day 27 request boundary](../docs/security/day-27-run-test-request-boundary.md).

The Runner-facing routes remain disabled unless
`PROOFLAYER_RUNNER_TOKEN` contains a 32–128 character base64url-style bearer
credential. When enabled, the local process exposes identity-bound lease,
acknowledgement, stage-update, and completion routes. The token is never logged;
the ephemeral Ed25519 public key is logged so a same-host PoC Runner can pin it.

Plain HTTP remains loopback-only on both sides. This mode must not be exposed to
the Windows VM network, a LAN, a customer environment, or the public internet.
The next milestone adds the Windows worker orchestration and an authenticated
transport suitable for the isolated VM boundary.

The initial API contract is documented in
[`docs/architecture/runner-control-plane-protocol-v0.1.md`](../docs/architecture/runner-control-plane-protocol-v0.1.md).
