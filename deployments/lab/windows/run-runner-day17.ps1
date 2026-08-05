#Requires -RunAsAdministrator
[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
$runnerPath = Join-Path $PSScriptRoot "prooflayer-runner-lab.exe"

if (-not (Test-Path -LiteralPath $runnerPath -PathType Leaf)) {
    throw "The fixed Runner lab binary is missing from the read-only media."
}

$rawResult = & $runnerPath
if ($LASTEXITCODE -ne 0) {
    throw "The fixed Runner lab handler failed with exit code $LASTEXITCODE."
}

$result = $rawResult | ConvertFrom-Json
if ($result.status -ne "passed") {
    throw "The fixed Runner lab handler did not return PASS."
}
if ($result.cleanup_status -ne "passed") {
    throw "The fixed Runner lab handler did not return cleanup PASS."
}

[ordered]@{
    Status = "PASS"
    CorrelationId = $result.correlation_id
    ScenarioId = $result.scenario_id
    ScenarioVersion = $result.scenario_version
    ExecutionLatencyMs = $result.latency_ms
    CleanupStatus = $result.cleanup_status
    AuditPath = "$env:ProgramData\ProofLayer\runner-audit.jsonl"
} | ConvertTo-Json
