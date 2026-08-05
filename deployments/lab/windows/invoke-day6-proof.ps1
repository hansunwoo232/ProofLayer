#Requires -RunAsAdministrator
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[A-Fa-f0-9]{32,128}$')]
    [string]$HecToken,

    [Parameter()]
    [ValidatePattern('^https://10\.0\.2\.100:8088/services/collector/event$')]
    [string]$HecUri = "https://10.0.2.100:8088/services/collector/event",

    [Parameter(Mandatory = $true)]
    [switch]$AllowLocalLabSelfSignedCertificate
)

$ErrorActionPreference = "Stop"
$channel = "Microsoft-Windows-Sysmon/Operational"
$correlationId = "PL-$([guid]::NewGuid().ToString('N').ToUpperInvariant())"
$executionStartedAt = [datetime]::UtcNow

$logInfo = Get-WinEvent -ListLog $channel
if (-not $logInfo.IsEnabled) {
    throw "Sysmon event channel is not enabled: $channel"
}

Start-Process -FilePath "cmd.exe" `
    -ArgumentList "/d", "/c", "echo $correlationId >NUL" `
    -Wait `
    -WindowStyle Hidden

$deadline = [datetime]::UtcNow.AddSeconds(15)
$event = $null
do {
    $event = Get-WinEvent -FilterHashtable @{
        LogName = $channel
        Id = 1
        StartTime = $executionStartedAt.AddSeconds(-2)
    } -ErrorAction SilentlyContinue |
        Where-Object { $_.Message -like "*$correlationId*" } |
        Select-Object -First 1

    if (-not $event) {
        Start-Sleep -Milliseconds 500
    }
} while (-not $event -and [datetime]::UtcNow -lt $deadline)

if (-not $event) {
    throw "Sysmon Event ID 1 was not found for correlation ID $correlationId"
}

$eventXml = [xml]$event.ToXml()
$eventData = @{}
foreach ($entry in $eventXml.Event.EventData.Data) {
    $eventData[[string]$entry.Name] = [string]$entry.'#text'
}

$requiredData = @("UtcTime", "Image", "CommandLine", "User")
foreach ($field in $requiredData) {
    if ([string]::IsNullOrWhiteSpace($eventData[$field])) {
        throw "Sysmon Event ID 1 is missing required field: $field"
    }
}

$endpointEventTime = [datetime]::Parse(
    $eventData["UtcTime"],
    [Globalization.CultureInfo]::InvariantCulture,
    [Globalization.DateTimeStyles]::AssumeUniversal
).ToUniversalTime()

$payload = [ordered]@{
    time = [double]([DateTimeOffset]$endpointEventTime).ToUnixTimeMilliseconds() / 1000
    host = [string]$eventXml.Event.System.Computer
    source = "prooflayer:windows-lab"
    sourcetype = "prooflayer:sysmon"
    index = "prooflayer_test"
    event = [ordered]@{
        schema_version = "1.0"
        correlation_id = $correlationId
        event_kind = "endpoint_process"
        provider = "Microsoft-Windows-Sysmon"
        event_id = [int]$event.Id
        record_id = [long]$event.RecordId
        endpoint_event_time = $endpointEventTime.ToString("o")
        observed_at = [datetime]::UtcNow.ToString("o")
        host = [ordered]@{
            name = [string]$eventXml.Event.System.Computer
        }
        process = [ordered]@{
            name = [IO.Path]::GetFileName($eventData["Image"])
            image = $eventData["Image"]
            command_line = $eventData["CommandLine"]
        }
        user = [ordered]@{
            name = $eventData["User"]
        }
    }
}

Add-Type -AssemblyName System.Net.Http
if (-not ("ProofLayerLocalLabCertificateValidator" -as [type])) {
    Add-Type -TypeDefinition @"
using System;
using System.Net.Http;
using System.Net.Security;
using System.Security.Cryptography.X509Certificates;

public static class ProofLayerLocalLabCertificateValidator
{
    public static readonly Func<HttpRequestMessage, X509Certificate2, X509Chain, SslPolicyErrors, bool>
        Callback = Validate;

    private static bool Validate(
        HttpRequestMessage message,
        X509Certificate2 certificate,
        X509Chain chain,
        SslPolicyErrors errors)
    {
        return true;
    }
}
"@ -ReferencedAssemblies "System.Net.Http"
}

$handler = [Net.Http.HttpClientHandler]::new()
$handler.UseProxy = $false
$handler.ServerCertificateCustomValidationCallback =
    [ProofLayerLocalLabCertificateValidator]::Callback

$client = [Net.Http.HttpClient]::new($handler)
try {
    $client.Timeout = [TimeSpan]::FromSeconds(30)
    $client.DefaultRequestHeaders.Authorization =
        [Net.Http.Headers.AuthenticationHeaderValue]::new("Splunk", $HecToken)

    $json = $payload | ConvertTo-Json -Depth 8 -Compress
    $content = [Net.Http.StringContent]::new(
        $json,
        [Text.Encoding]::UTF8,
        "application/json"
    )
    try {
        $response = $client.PostAsync($HecUri, $content).GetAwaiter().GetResult()
        $responseBody = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult()
    }
    catch {
        $exceptionMessages = [Collections.Generic.List[string]]::new()
        $currentException = $_.Exception
        while ($null -ne $currentException) {
            $exceptionMessages.Add($currentException.Message)
            $currentException = $currentException.InnerException
        }
        throw "HEC request failed: $($exceptionMessages -join ' -> ')"
    }

    if (-not $response.IsSuccessStatusCode) {
        throw "HEC returned HTTP $([int]$response.StatusCode): $responseBody"
    }

    $hecResult = $responseBody | ConvertFrom-Json
    if ($hecResult.code -ne 0) {
        throw "HEC rejected the event: $responseBody"
    }
}
finally {
    $client.Dispose()
    $handler.Dispose()
}

$completedAt = [datetime]::UtcNow
[ordered]@{
    Status = "PASS"
    CorrelationId = $correlationId
    EventId = [int]$event.Id
    RecordId = [long]$event.RecordId
    EndpointEventTimeUtc = $endpointEventTime.ToString("o")
    HecAcceptedAtUtc = $completedAt.ToString("o")
    EndpointToHecAcceptedLatencyMs = [math]::Round(
        ($completedAt - $endpointEventTime).TotalMilliseconds,
        0
    )
} | ConvertTo-Json
