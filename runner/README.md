# Runner

The Runner is a Windows agent that executes only signed, allowlisted, safe
scenarios. The first PoC implementation language is Go. Arbitrary command or
PowerShell execution is excluded by design.

The Runner–Control Plane boundary is defined in
[`docs/architecture/runner-control-plane-protocol-v0.1.md`](../docs/architecture/runner-control-plane-protocol-v0.1.md).

## Current technical MVP

The current skeleton provides:

- A fixed built-in catalog containing `windows-process-marker@0.1.0`,
  `windows-registry-run-key-canary@0.1.0`, and
  `windows-scheduled-task-canary@0.1.0`
- Fail-closed lookup for unknown scenarios and versions
- A host- and environment-bound Runner identity model
- Read-only `version`, `catalog`, and `self-check` commands
- Cryptographically random canonical correlation ID generation
- Fail-closed execution-request validation and approved runtime limits
- Append-only, local JSONL audit event recording with restricted permissions
- Mandatory cleanup execution and independent artifact-absence verification
- Versioned `1.0` JSON execution results with one scenario-independent shape

The production CLI deliberately has no general execution command.

The separate process-marker, Registry, and Scheduled Task lab entry points are
temporary isolated-lab harnesses for their fixed handlers. They accept no
arguments and are not production Runner entry points.

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
