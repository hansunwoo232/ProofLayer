# ProofLayer JSON Contracts — v1

These JSON Schema contracts are the language-neutral interface between
ProofLayer components.

## Contracts

- `scenario.schema.json`: approved scenario metadata and safe action type
- `create-test-job-request.schema.json`: bounded operator request for one
  allowlisted scenario on one registered lab host
- `runner-execution-result.schema.json`: versioned result emitted by the Runner
- `test-job.schema.json`: short-lived signed execution authorization
- `test-job-receipt.schema.json`: idempotent queue acceptance response
- `test-run.schema.json`: stage-level execution and observation result

## Validate examples

From the repository root:

```bash
node tools/validate-contract-examples.mjs
```

The dependency-free validator checks JSON syntax and the security-critical
invariants used by the process-marker, Registry Run Key canary, Scheduled Task
canary, and their versioned Runner results. It also proves that all three Runner
result examples have the same field shape. CI should later add a full JSON
Schema Draft 2020-12 validator.
