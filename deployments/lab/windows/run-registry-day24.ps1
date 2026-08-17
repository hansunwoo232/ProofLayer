#Requires -RunAsAdministrator
[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
$runnerPath = Join-Path $PSScriptRoot "prooflayer-registry-canary-lab.exe"
$runKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run"

if (-not (Test-Path -LiteralPath $runnerPath -PathType Leaf)) {
    throw "The fixed Registry canary lab binary is missing from the read-only media."
}

$rawResult = & $runnerPath
if ($LASTEXITCODE -ne 0) {
    throw "The fixed Registry canary handler failed with exit code $LASTEXITCODE."
}

$result = $rawResult | ConvertFrom-Json
if ($result.status -ne "passed" -or $result.cleanup_status -ne "passed") {
    throw "The Registry canary did not return execution PASS and cleanup PASS."
}

$valueName = "ProofLayer_" + $result.correlation_id.Substring(3)
$artifactAbsent = $false
try {
    Get-ItemPropertyValue -LiteralPath $runKey -Name $valueName -ErrorAction Stop | Out-Null
}
catch [System.Management.Automation.PSArgumentException] {
    $artifactAbsent = $true
}
catch [System.Management.Automation.ItemNotFoundException] {
    $artifactAbsent = $true
}

if (-not $artifactAbsent) {
    throw "Independent verification found the Registry canary after cleanup."
}

[ordered]@{
    Status = "PASS"
    CorrelationId = $result.correlation_id
    ScenarioId = $result.scenario_id
    ScenarioVersion = $result.scenario_version
    ExecutionAndCleanupLatencyMs = $result.latency_ms
    CleanupStatus = $result.cleanup_status
    IndependentArtifactAbsence = "PASS"
    RegistryPath = "HKCU\Software\Microsoft\Windows\CurrentVersion\Run"
    AuditPath = "$env:ProgramData\ProofLayer\runner-audit.jsonl"
} | ConvertTo-Json
