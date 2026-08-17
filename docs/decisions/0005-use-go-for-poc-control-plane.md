# ADR-0005 — Use Go for the PoC Control Plane

- **Date:** 2026-08-17
- **Status:** Accepted
- **Owner:** Founding team

## Context

The Day 27 Control Plane needs a small local HTTP boundary, a concurrency-safe
job queue, Ed25519 job authorization, strict request parsing, and tests that do
not depend on external infrastructure. The initial design left the language
choice open between Go and FastAPI.

## Decision

Use Go and the standard library for the PoC Control Plane. Keep it as a separate
Go module under `control-plane/`. The first process binds only to loopback and
uses an in-memory queue; persistent storage and production identity are not
implied by this decision.

## Consequences

- The queue, HTTP boundary, signing, timeouts, and concurrency behavior use one
  dependency-free runtime.
- The Runner and Control Plane can share language-level implementation habits
  without becoming one trust boundary.
- The PoC loses queued jobs and its ephemeral signing key on restart.
- A production deployment still requires durable storage, managed signing
  keys, authentication/RBAC, TLS, and outbound Runner transport.

## Validation

- Concurrent duplicate submissions produce one queued job.
- Signed jobs verify and are bound to one environment and host.
- Expired jobs cannot be leased.
- Strict HTTP request and request-integrity negative tests pass.
