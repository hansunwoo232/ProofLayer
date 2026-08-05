# Windows-to-Splunk Lab Reproduction Runbook

## Scope

This runbook reproduces the local ProofLayer Windows 11 Arm64 → Sysmon → Splunk
proof on the authorized Apple Silicon development host. It is not a customer
deployment guide.

## 1. Start and validate Splunk

```bash
cd deployments/splunk
./setup.sh
./wait-until-ready.sh
./create-test-index.sh
./configure-hec.sh
./smoke-test.sh
```

Acceptance:

- Container state is `running healthy`.
- Authenticated management API returns HTTP 200.
- `prooflayer_test` exists.
- The synthetic smoke event is returned by search.

## 2. Build read-only Day 6 media

```bash
cd ../lab
./build-day6-iso.sh
```

The generated ISO is outside Git and contains a dedicated local HEC token.
Record its SHA-256 for the run, never share it, and rotate the token when the
experiment is retired.

The builder also verifies the pinned VirtIO-Win package and copies only the
Microsoft-signed Windows 11 Arm64 NetKVM files documented in
`deployments/lab/VIRTIO-NETKVM.md`.

## 3. Start the isolated Windows lab

Set `LAB_NETWORK_MODE="siem"` in ignored `lab.env`, then:

```bash
./labctl.sh preflight
./labctl.sh tools
```

`siem` mode keeps `restrict=on` and adds only this explicit path through a
fixed command-backed relay:

```text
guest 10.0.2.100:8088 → host 127.0.0.1:8088 → Splunk HEC
```

## 4. Install the Arm64 network driver when required

On a clean Windows checkpoint, open Administrator PowerShell:

```powershell
Set-ExecutionPolicy -Scope Process Bypass -Force
$installer = Get-Volume | Where-Object DriveType -eq 'CD-ROM' | ForEach-Object { Get-Item "$($_.DriveLetter):\install-netkvm.ps1" -ErrorAction SilentlyContinue } | Select-Object -First 1
& $installer.FullName
```

Acceptance:

- Script status is `PASS`.
- Adapter is `Red Hat VirtIO Ethernet Adapter`.
- Link status is `Up`.
- Driver provider is Microsoft Windows Hardware Compatibility Publisher.

This installation is persistent and does not need to be repeated after it has
been captured in the local checkpoint.

## 5. Execute the safe proof

Open Administrator PowerShell in Windows:

```powershell
Set-ExecutionPolicy -Scope Process Bypass -Force
$media = Get-Volume | Where-Object DriveType -eq 'CD-ROM' | ForEach-Object { Get-Item "$($_.DriveLetter):\run-day6.ps1" -ErrorAction SilentlyContinue } | Select-Object -First 1
& $media.FullName
```

Acceptance:

- One canonical `PL-` correlation ID is generated.
- A harmless `cmd.exe` marker process completes.
- Sysmon Event ID 1 contains the exact ID.
- HEC accepts the minimum structured event.
- The PowerShell result is `PASS`.

Before the first proof in a session, the HEC health path can be checked with
the dedicated local token loaded by `run-day6.ps1`. Never print the token.

## 6. Verify in Splunk

After the Windows script completes:

```bash
cd deployments/splunk
./verify-latest-windows-event.sh
./verify-windows-event-series.sh
```

The latest-event verifier searches by canonical correlation ID, confirms Event
ID 1 and required parsed fields, calculates `_indextime - _time`, and stores
ignored JSON evidence. After three runs, the series verifier requires three
unique IDs and records minimum, average, and maximum ingestion latency.

## 7. Close and preserve

1. Shut down Windows normally.
2. Confirm the QEMU process exits.
3. Create a checkpoint only while the VM is stopped.
4. Stop Splunk when it is no longer needed.
5. Rotate the HEC token before sharing or rebuilding local media for another
   environment.

## Troubleshooting boundaries

- Never switch to `nat` to solve HEC connectivity.
- Never disable the host firewall.
- Never expose Splunk ports beyond loopback.
- Never paste the Splunk administrator password into Windows.
- Never commit `.env`, ISO, VM disk, evidence, or generated secrets.
