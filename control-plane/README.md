# Control plane

The Control Plane owns Runner registration, scheduling, correlation, audit
events, and result APIs. The first technical decision gate will compare Go and
FastAPI for PoC speed, security boundaries, and deployment simplicity.

The initial API contract is documented in
[`docs/architecture/runner-control-plane-protocol-v0.1.md`](../docs/architecture/runner-control-plane-protocol-v0.1.md).
