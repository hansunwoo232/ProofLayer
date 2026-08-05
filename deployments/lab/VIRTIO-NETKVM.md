# VirtIO NetKVM Verification

The ProofLayer Windows 11 Arm64 lab uses the NetKVM driver from the official
VirtIO-Win `0.1.285` pre-WHQL package published by the Fedora VirtIO-Win project.

## Source

```text
https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/virtio-win-pkg-scripts-input/virtio-win-0.1.285-1/virtio-win-prewhql-0.1.zip
```

Local package name:

```text
work/prooflayer-lab/virtio-win-prewhql-0.1.285.zip
```

Package SHA-256:

```text
db3b687b2863efc874e31520f3aed07fc106b1cad295dfa0a86f96855b0fc8ab
```

## Included driver

Only these `Win11/ARM64` NetKVM files are copied into the ignored Day 6 ISO:

- `netkvm.cat`
- `netkvm.inf`
- `netkvm.sys`
- `netkvmco.exe`
- `netkvmp.exe`

`install-netkvm.ps1` verifies every file hash before installation. Windows
reported the catalog signer as Microsoft Windows Hardware Compatibility
Publisher and the installed adapter as Red Hat VirtIO Ethernet Adapter.

The package and generated ISO must not be committed or redistributed from this
repository.
