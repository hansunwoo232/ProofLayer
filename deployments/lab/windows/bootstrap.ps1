#Requires -RunAsAdministrator
[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
$LabRoot = "C:\ProofLayerLab"
$ConfigPath = Join-Path $PSScriptRoot "sysmon-lab.xml"
$BundledSysmonPath = Join-Path $PSScriptRoot "Sysmon64a.exe"

if (-not (Test-Path $ConfigPath)) {
    throw "Sysmon configuration not found: $ConfigPath"
}

if ((Get-TimeZone).Id -ne "UTC") {
    Set-TimeZone -Id "UTC"
    throw "The lab time zone was changed to UTC. Restart Windows and rerun bootstrap before validating telemetry."
}

New-Item -ItemType Directory -Path $LabRoot -Force | Out-Null

$SysmonPath = $null
if (Test-Path $BundledSysmonPath) {
    $SysmonPath = $BundledSysmonPath
}
elseif (Get-Command "sysmon" -ErrorAction SilentlyContinue) {
    $SysmonPath = "sysmon"
}
else {
    $feature = Get-WindowsOptionalFeature -Online -FeatureName "Sysmon" -ErrorAction SilentlyContinue
    if (-not $feature -or $feature.State -ne "Enabled") {
        Enable-WindowsOptionalFeature -Online -FeatureName "Sysmon" -NoRestart
    }
    $SysmonPath = "sysmon"
}

$existingServices = Get-Service -Name "Sysmon*" -ErrorAction SilentlyContinue
if (-not $existingServices) {
    & $SysmonPath -accepteula -i $ConfigPath
}
else {
    & $SysmonPath -c $ConfigPath
}

$channel = "Microsoft-Windows-Sysmon/Operational"
$logInfo = Get-WinEvent -ListLog $channel
if (-not $logInfo.IsEnabled) {
    throw "Sysmon event channel is not enabled: $channel"
}

$marker = "PL-$([guid]::NewGuid().ToString('N').ToUpperInvariant())"
Start-Process -FilePath "cmd.exe" -ArgumentList "/c", "echo $marker" -Wait -WindowStyle Hidden

Start-Sleep -Seconds 2
$event = Get-WinEvent -FilterHashtable @{
    LogName = $channel
    Id = 1
    StartTime = (Get-Date).AddMinutes(-2)
} | Where-Object { $_.Message -like "*$marker*" } | Select-Object -First 1

if (-not $event) {
    throw "Sysmon process creation event was not found for marker $marker"
}

[pscustomobject]@{
    Status = "PASS"
    CorrelationId = $marker
    EventId = $event.Id
    RecordId = $event.RecordId
    TimestampUtc = $event.TimeCreated.ToUniversalTime().ToString("o")
} | ConvertTo-Json
