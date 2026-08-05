#Requires -RunAsAdministrator
[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
$driverRoot = Join-Path $PSScriptRoot "NetKVM\Win11\ARM64"
$driverInf = Join-Path $driverRoot "netkvm.inf"

$expectedHashes = [ordered]@{
    "netkvm.cat" = "7185AE6C8C6D31D047C77B167E740B7FE3AA676A89B853783CDC47FD889D51C5"
    "netkvm.inf" = "1AAF16AF51EF4697503E7C261F318A9414DC726BE5250F09CC64D558F46138FC"
    "netkvm.sys" = "211340B8128145BAE8890A4FAB27B625C23E032CA65BE984B524E1B0D4B54F0F"
    "netkvmco.exe" = "346C35078DC7ADF0C9521243FF817DA5E882AF668155E7D736FA4BCCDC0B04F9"
    "netkvmp.exe" = "98B239DC9E4080CC21211582F05AB1D7F6AC74D867980B183C3F8F5AE9D5DC5A"
}

foreach ($entry in $expectedHashes.GetEnumerator()) {
    $path = Join-Path $driverRoot $entry.Key
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Missing NetKVM driver file: $path"
    }

    $actualHash = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash
    if ($actualHash -ne $entry.Value) {
        throw "SHA-256 mismatch for NetKVM driver file: $($entry.Key)"
    }
}

$catalogSignature = Get-AuthenticodeSignature -FilePath (Join-Path $driverRoot "netkvm.cat")
if ($catalogSignature.Status -ne "Valid") {
    throw "NetKVM catalog signature is not valid: $($catalogSignature.Status)"
}

$installOutput = & pnputil.exe /add-driver $driverInf /install 2>&1
if ($LASTEXITCODE -ne 0) {
    throw "NetKVM installation failed: $($installOutput -join ' ')"
}

& pnputil.exe /scan-devices | Out-Null
Start-Sleep -Seconds 3

$adapter = Get-NetAdapter |
    Where-Object { $_.InterfaceDescription -like "*VirtIO*" } |
    Select-Object -First 1

if (-not $adapter) {
    throw "The VirtIO network adapter was not found after driver installation."
}

[ordered]@{
    Status = "PASS"
    AdapterName = $adapter.Name
    InterfaceDescription = $adapter.InterfaceDescription
    LinkStatus = [string]$adapter.Status
    DriverProvider = $catalogSignature.SignerCertificate.Subject
} | ConvertTo-Json
