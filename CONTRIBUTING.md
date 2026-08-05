# ProofLayer contribution guidelines

## Working model

- Keep `main` demonstrable at all times.
- Keep changes small, single-purpose, and reversible.
- Record material architecture decisions under `docs/decisions/`.
- Update `docs/product/mvp-scope.md` before changing product scope.
- Never commit customer data, credentials, tokens, raw events, or local lab state.

## Definition of done

Work is complete only when its acceptance criteria pass, the success path and at
least one failure path are tested, security impact is reviewed, required cleanup
is automatic and verified, operator errors are understandable, and related
documentation is current.

## Code and API rules

- Runner: Go. Control plane: Go or Python/FastAPI after a PoC decision.
  Dashboard: TypeScript.
- External API boundaries require input schemas and explicit versions.
- Identifiers must be unpredictable and timestamps must be stored in UTC.
- Logs must not contain secrets, customer content, or full command lines by
  default.
- Retries must be bounded and idempotent.
- Defaults must be secure; any weakening option must be explicit.

## Branches and commits

- Suggested branch pattern: `type/short-description`.
- Types: `feat`, `fix`, `docs`, `test`, `security`, and `chore`.
- Commit messages should describe the outcome of the change.

## Security invariants

- The Runner uses outbound connections; no inbound customer connection is
  opened.
- There is no arbitrary shell or PowerShell execution API.
- A test package must pass integrity and signature verification before execution.
- The kill switch and execution timeout cannot be bypassed.
- Any state-changing behavior requires a unique canary and verified cleanup.
- Tests never run on unauthorized or out-of-scope hosts.
