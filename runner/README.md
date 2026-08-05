# Runner

The Runner is a Windows agent that executes only signed, allowlisted, safe
scenarios. The first PoC implementation language is Go. Arbitrary command or
PowerShell execution is excluded by design.

The Runner–Control Plane boundary is defined in
[`docs/architecture/runner-control-plane-protocol-v0.1.md`](../docs/architecture/runner-control-plane-protocol-v0.1.md).

## Day 15 skeleton

The current skeleton provides:

- A fixed built-in catalog containing only `windows-process-marker@0.1.0`
- Fail-closed lookup for unknown scenarios and versions
- A host- and environment-bound Runner identity model
- Read-only `version`, `catalog`, and `self-check` commands
- Cryptographically random canonical correlation ID generation
- Fail-closed execution-request validation and approved runtime limits
- Append-only, local JSONL audit event recording with restricted permissions

It deliberately has no execution command yet.

The separate `prooflayer-runner-lab` entry point is a temporary isolated-lab
harness for the one fixed built-in handler. It accepts no arguments and is not
the production Runner entry point.

```text
go run ./cmd/prooflayer-runner self-check
go run ./cmd/prooflayer-runner catalog
go run ./cmd/prooflayer-runner new-correlation
go test ./...
```

Build the Windows ARM64 binary used by the current lab with:

```text
GOOS=windows GOARCH=arm64 go build ./cmd/prooflayer-runner
```
