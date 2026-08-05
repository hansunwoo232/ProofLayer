# ProofLayer Tools ISO

**Generated:** August 2, 2026  
**Purpose:** Offline Sysmon bootstrap media for the isolated Windows lab

## Source

- Sysmon download: `https://download.sysinternals.com/files/Sysmon.zip`
- Sysmon version: `15.21`
- Architecture used: Arm64 (`Sysmon64a.exe`)

## SHA-256

| Artifact | SHA-256 |
|---|---|
| Official `Sysmon.zip` download | `6d48089c7fae14944c82b06767b79ccba3cc26d13218a4227ed28c90f80d0f0e` |
| Generated `prooflayer-tools.iso` | `8efb0da7025bb8e389ea333dc5dc07c638e3ff5ed7c01da50522753187bfedb2` |

## Included files

- `Sysmon64a.exe`
- `bootstrap.ps1`
- `sysmon-lab.xml`
- Microsoft Sysmon EULA

The ISO is attached to the guest as read-only media. It contains no customer
data, credentials, secrets, or writable host share.
