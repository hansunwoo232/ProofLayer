# Control Plane

The Control Plane owns Runner registration, scheduling, correlation, audit
events, and result APIs. ADR-0005 selects Go for the technical PoC.

## Day 27 local process

From the `control-plane` directory:

```bash
go run ./cmd/prooflayer-control-plane
```

Open `http://127.0.0.1:8787`. The process serves the result surface and accepts
one fixed, allowlisted `Run Test` request. It binds only to loopback.

The current implementation provides:

- exact-origin and double-submit CSRF validation;
- strict, size-limited JSON parsing;
- server-side environment, host, scenario, and version binding;
- duplicate-safe idempotency handling;
- a bounded concurrency-safe queue;
- two-minute Ed25519-signed Test Jobs; and
- a one-time host-bound lease interface for the next Runner transport step.

This is a local-only process. Its queue and signing key are ephemeral, it has no
operator authentication or durable audit store, and it must not be exposed to
a LAN, customer environment, or public network. See the
[Day 27 request boundary](../docs/security/day-27-run-test-request-boundary.md).

The initial API contract is documented in
[`docs/architecture/runner-control-plane-protocol-v0.1.md`](../docs/architecture/runner-control-plane-protocol-v0.1.md).
