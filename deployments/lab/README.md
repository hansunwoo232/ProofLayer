# ProofLayer Windows Lab

This lab provides an isolated Windows 11 Arm64 VM on Apple Silicon using QEMU.
It is intended only for safe ProofLayer telemetry experiments.

## Current host status

Validated through August 5, 2026:

- Host architecture: Apple Silicon (`arm64`)
- QEMU: `10.0.3`
- QEMU Arm64 UEFI firmware: available
- VirtualBox: installed but not responsive for guest-type discovery
- Windows 11 25H2 English Arm64 ISO: available and SHA-256 verified
- `swtpm`: `0.10.1`

The VM disk, UEFI state, virtual TPM, and installation media are ready. Windows
11 Pro is installed and validated with disk-only boot. Sysmon v15.21 Arm64 is
installed and the first correlation-marker process event has been verified.

## Security boundaries

- The default mode is `isolated`.
- Internet/NAT mode requires `ALLOW_INTERNET=YES`.
- No host directory is shared with the guest.
- The VM state is stored outside the Git repository.
- No corporate VPN, production SIEM, or customer network may be connected.
- Snapshots must be taken before installing a Runner or executing a scenario.

## Required resources

- 8 GB guest RAM (12 GB recommended if the host permits)
- 4 virtual CPUs
- 80 GB sparse QCOW2 disk
- Windows 11 Arm64 ISO from Microsoft
- `swtpm` for a virtual TPM 2.0 device

Microsoft provides Windows 11 Arm64 ISO images for creating VMs on Apple
Silicon. Use one of these official sources:

- https://learn.microsoft.com/en-us/windows/arm/iso
- https://www.microsoft.com/en-us/software-download/windows11arm64
- https://www.microsoft.com/en-us/evalcenter/evaluate-windows-11-enterprise

Do not commit installation media or VM disks.

The downloaded ISO verification result is recorded in
[ISO-VERIFICATION.md](ISO-VERIFICATION.md).

The offline Sysmon media contents and hashes are recorded in
[TOOLS-ISO.md](TOOLS-ISO.md).

The Windows 11 Arm64 network driver source, package hash, and trust result are
recorded in [VIRTIO-NETKVM.md](VIRTIO-NETKVM.md).

## Quick start

```bash
cd deployments/lab
cp lab.env.example lab.env
./labctl.sh preflight
./labctl.sh prepare
```

Set an absolute ISO path in `lab.env`, install `swtpm`, then:

```bash
./labctl.sh install
```

After Windows installation:

```bash
./labctl.sh start
```

Start Windows with the read-only ProofLayer tools ISO attached:

```bash
./labctl.sh tools
```

For the Day 6 endpoint-to-SIEM proof, build the ignored local media after HEC
configuration:

```bash
./build-day6-iso.sh
./labctl.sh tools
```

The Day 6 ISO contains a local-only HEC token so the isolated guest can submit
synthetic telemetry. It must never be committed, shared, or used outside this
lab. Rotate the HEC token when the experiment is retired.

Create and list lab checkpoints while the VM is stopped:

```bash
./labctl.sh checkpoint checkpoint-name
./labctl.sh checkpoints
```

## Installer boot note

On the validated QEMU/HVF host, the firmware may open the UEFI Interactive
Shell instead of starting Windows Setup automatically. Start the verified
Arm64 installer with:

```text
fs0:\efi\boot\bootaa64.efi
```

The installation ISO must remain attached as USB storage so Windows PE can
read the installation source without an additional storage driver.

## Validated baseline

The `clean-windows-11-pro` checkpoint contains the clean Windows baseline
before Sysmon installation. The Sysmon bootstrap produced a validated Event ID
1 with correlation ID `PL-TEST-BOOTSTRAP-7E31F075A720` at
`2026-08-01T21:57:31.9191350Z`.

The `sysmon-baseline` checkpoint contains the validated Windows and Sysmon
state immediately after the successful correlation-marker test.

The `day-10-windows-splunk-proof` checkpoint contains the Microsoft-signed
Arm64 NetKVM driver and the Windows state after three successful
Windows → Sysmon → HEC → Splunk runs. UEFI and TPM state are preserved beside
the QCOW2 snapshot in the ignored local lab directory.

The `day-21-runner-observer-proof` checkpoint preserves the Runner, local
Sysmon Observer, exact Splunk correlation, field-validation, and detection-plan
proof completed before the Day 24 Registry canary.

The `day-24-registry-cleanup-proof` checkpoint preserves the clean Windows,
UEFI, and TPM state after the Registry canary returned execution PASS, cleanup
PASS, and independent artifact-absence PASS.

## Day 24 Registry Run Key canary

Build the ignored read-only Runner media and attach it to the isolated VM:

```bash
./build-runner-iso.sh
LAB_TOOLS_ISO_OVERRIDE="../../work/prooflayer-lab/prooflayer-runner.iso" ./labctl.sh tools
```

Inside an elevated PowerShell window, locate the CD-ROM and run the fixed
wrapper:

```powershell
Set-ExecutionPolicy -Scope Process Bypass -Force
$media = Get-Volume | Where-Object DriveType -eq 'CD-ROM' | ForEach-Object { Get-Item "$($_.DriveLetter):\run-registry-day24.ps1" -ErrorAction SilentlyContinue } | Select-Object -First 1
& $media.FullName
```

PASS requires both the Runner cleanup result and an independent PowerShell
query to confirm that the correlation-bound HKCU Run value is absent.

## Day 25 Scheduled Task canary

Rebuild the ignored read-only Runner media and attach it to the isolated VM:

```bash
./build-runner-iso.sh
LAB_TOOLS_ISO_OVERRIDE="../../work/prooflayer-lab/prooflayer-runner.iso" ./labctl.sh tools
```

Inside an elevated PowerShell window, locate the CD-ROM and run the fixed
wrapper:

```powershell
Set-ExecutionPolicy -Scope Process Bypass -Force
$media = Get-Volume | Where-Object DriveType -eq 'CD-ROM' | ForEach-Object { Get-Item "$($_.DriveLetter):\run-scheduled-task-day25.ps1" -ErrorAction SilentlyContinue } | Select-Object -First 1
& $media.FullName
```

PASS requires the Runner cleanup result, an independent Task Scheduler COM
lookup, and an independent task artifact file check to confirm absence.

The `day-25-scheduled-task-cleanup-proof` checkpoint preserves the clean
Windows, UEFI, and TPM state after all three checks returned PASS.

## Network modes

| Mode | Behavior | Intended use |
|---|---|---|
| `offline` | No virtual NIC | Highest-isolation scenario testing |
| `isolated` | NIC exists; outbound and host access restricted | Default VM operation |
| `siem` | Guest remains restricted; only `10.0.2.100:8088` forwards to host HEC | Endpoint-to-SIEM lab validation |
| `nat` | Guest can reach the internet | Windows Update only; explicit opt-in |

Never use `nat` while a ProofLayer scenario is executing.

The `siem` mode uses QEMU's explicit command-backed `guestfwd` rule. Each guest
connection to `10.0.2.100:8088` starts the host's `/usr/bin/nc` with the fixed
destination `127.0.0.1:8088`. It does not grant general host or internet
access. The guest-visible HEC address is `https://10.0.2.100:8088`.

The Windows 11 Arm64 guest requires the Microsoft-signed NetKVM driver included
in the ignored Day 6 media. On a clean checkpoint, run `install-netkvm.ps1` once
as Administrator before testing HEC connectivity.

## Windows bootstrap

After the OS is installed, copy the files in `windows/` into the VM and run
PowerShell as Administrator:

```powershell
Set-ExecutionPolicy -Scope Process Bypass
.\bootstrap.ps1
```

The bootstrap script enables Sysmon, applies the lab-only configuration, and
creates the local `C:\ProofLayerLab` workspace.

## Day 3 acceptance status

| Acceptance item | Status |
|---|---|
| Hypervisor path selected | PASS — QEMU/HVF |
| Isolated network design | PASS |
| Repeatable VM scripts | PASS |
| Sparse VM disk prepared | PASS |
| Windows ISO available | PASS — Microsoft SHA-256 exact match |
| Virtual TPM available | PASS — `swtpm 0.10.1` |
| Windows installer launched | PASS |
| Windows installation completed | PASS — Windows 11 Pro |
| Disk-only boot | PASS |
| Clean Windows checkpoint | PASS — `clean-windows-11-pro` |
| Sysmon Arm64 installation | PASS — v15.21 |
| Correlation-marker validation | PASS — Event ID 1 |
| Sysmon baseline checkpoint | PASS — `sysmon-baseline` |

## Day 6–10 acceptance status

| Acceptance item | Status |
|---|---|
| Microsoft-signed Arm64 NetKVM installation | PASS |
| Restricted HEC health path | PASS |
| Safe process-marker execution | PASS — three runs |
| Sysmon Event ID 1 correlation | PASS — three unique IDs |
| HEC authenticated acceptance | PASS — three runs |
| Splunk required-field extraction | PASS |
| Least-privilege observer search | PASS |
| Observer `_internal` negative access test | PASS |
| Ingestion latency series | PASS — 1365–2677 ms |
| Preserved final lab checkpoint | PASS — `day-10-windows-splunk-proof` |
