#Requires -RunAsAdministrator
[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
$executable = Get-Volume |
    Where-Object DriveType -eq "CD-ROM" |
    ForEach-Object {
        Get-Item "$($_.DriveLetter):\prooflayer-day30-lab.exe" -ErrorAction SilentlyContinue
    } |
    Select-Object -First 1

if ($null -eq $executable) {
    throw "prooflayer-day30-lab.exe was not found on mounted CD-ROM media."
}

& $executable.FullName
if ($LASTEXITCODE -ne 0) {
    throw "ProofLayer Day 30 pipeline failed with exit code $LASTEXITCODE."
}
