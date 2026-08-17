#Requires -RunAsAdministrator
[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
$runnerPath = Join-Path $PSScriptRoot "prooflayer-scheduled-task-canary-lab.exe"

if (-not (Test-Path -LiteralPath $runnerPath -PathType Leaf)) {
    throw "The fixed Scheduled Task canary lab binary is missing from the read-only media."
}

$rawResult = & $runnerPath
if ($LASTEXITCODE -ne 0) {
    throw "The fixed Scheduled Task canary handler failed with exit code $LASTEXITCODE."
}

$result = $rawResult | ConvertFrom-Json
if ($result.status -ne "passed" -or $result.cleanup_status -ne "passed") {
    throw "The Scheduled Task canary did not return execution PASS and cleanup PASS."
}

$taskName = "ProofLayer_" + $result.correlation_id.Substring(3)
$scheduler = New-Object -ComObject "Schedule.Service"
$scheduler.Connect()
$rootFolder = $scheduler.GetFolder("\")
$matchingTasks = @($rootFolder.GetTasks(0) | Where-Object Name -eq $taskName)
if ($matchingTasks.Count -ne 0) {
    throw "Independent Task Scheduler verification found the canary after cleanup."
}

$taskFile = Join-Path $env:SystemRoot "System32\Tasks\$taskName"
if (Test-Path -LiteralPath $taskFile) {
    throw "Independent filesystem verification found the Scheduled Task artifact after cleanup."
}

[ordered]@{
    Status = "PASS"
    CorrelationId = $result.correlation_id
    ScenarioId = $result.scenario_id
    ScenarioVersion = $result.scenario_version
    ExecutionAndCleanupLatencyMs = $result.latency_ms
    CleanupStatus = $result.cleanup_status
    IndependentSchedulerAbsence = "PASS"
    IndependentTaskFileAbsence = "PASS"
    TaskName = $taskName
    Trigger = "ONLOGON"
    RunLevel = "LIMITED"
    AuditPath = "$env:ProgramData\ProofLayer\runner-audit.jsonl"
} | ConvertTo-Json
