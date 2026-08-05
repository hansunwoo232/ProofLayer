#Requires -RunAsAdministrator
[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
$secretPath = Join-Path $PSScriptRoot "prooflayer-lab-secret.ps1"
$proofPath = Join-Path $PSScriptRoot "invoke-day6-proof.ps1"

if (-not (Test-Path $secretPath)) {
    throw "Local lab secret file is missing from the read-only media."
}
if (-not (Test-Path $proofPath)) {
    throw "Day 6 proof script is missing from the read-only media."
}

. $secretPath
if ([string]::IsNullOrWhiteSpace($env:PROOFLAYER_HEC_TOKEN)) {
    throw "The local HEC token was not loaded."
}

& $proofPath `
    -HecToken $env:PROOFLAYER_HEC_TOKEN `
    -AllowLocalLabSelfSignedCertificate

Remove-Item Env:\PROOFLAYER_HEC_TOKEN -ErrorAction SilentlyContinue
