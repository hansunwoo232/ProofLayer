# Control Plane

The Control Plane owns Runner registration, scheduling, correlation, audit
events, and result APIs. ADR-0005 selects Go for the technical PoC.

## Day 30 local process

From the `control-plane` directory:

```bash
go run ./cmd/prooflayer-control-plane
```

Open `http://127.0.0.1:8787`. The process serves the result surface and accepts
one fixed, allowlisted `Run Test` request. It binds only to loopback.

To enable the Day 31 local sign-in boundary, provide a 12–128 character
bootstrap password before starting the process:

```bash
export PROOFLAYER_LOCAL_ADMIN_EMAIL="admin@prooflayer.local"
export PROOFLAYER_LOCAL_ADMIN_PASSWORD="use-a-local-development-secret"
go run ./cmd/prooflayer-control-plane
```

The password is converted to an Argon2id verifier during startup and then
removed from the process environment. It is never returned to or stored by the
browser. Local authentication is opt-in so the isolated Day 30 acceptance flow
remains reproducible; it must be enabled for operator-authenticated MVP use.

The browser, lifecycle, and Runner worker boundary provide:

- exact-origin and double-submit CSRF validation;
- workspace-bound local operator sessions with idle and absolute expiry;
- digest-only server-side session storage and immediate logout revocation;
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

This is a local-only process. Its queue, local user, and sessions are ephemeral,
and it has no durable audit store. It must not be exposed to a LAN, customer
environment, or public network. See the
[Day 27 request boundary](../docs/security/day-27-run-test-request-boundary.md).

The local identity boundary is documented in
[`docs/architecture/local-user-workspace-and-session-model-v0.1.md`](../docs/architecture/local-user-workspace-and-session-model-v0.1.md).

The Runner-facing routes remain disabled unless
`PROOFLAYER_RUNNER_TOKEN` contains a 32–128 character base64url-style bearer
credential. When enabled, the local process exposes identity-bound lease,
acknowledgement, stage-update, and completion routes. The token is never logged;
the Ed25519 public key is logged so a PoC Runner can pin it. Set a base64url
encoded 32-byte `PROOFLAYER_SIGNING_SEED` to keep this key stable across local
restarts; without it, the process generates an ephemeral key.

Plain HTTP remains loopback-only. When both `PROOFLAYER_RUNNER_TLS_CERT` and
`PROOFLAYER_RUNNER_TLS_KEY` are set, the same Runner API also listens on
`127.0.0.1:8788` using TLS. The QEMU lab exposes that listener only through the
fixed `10.0.2.100:8788` guest forward. This isolated-lab mode must not be exposed
to a LAN, customer environment, or the public internet.

The initial API contract is documented in
[`docs/architecture/runner-control-plane-protocol-v0.1.md`](../docs/architecture/runner-control-plane-protocol-v0.1.md).
