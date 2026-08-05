# Runner Negative Capability List

The MVP Runner is intentionally unable to perform the following actions. A
feature request that crosses this boundary requires a new ADR, threat-model
review, and explicit founder approval.

| Capability the Runner must not have | Enforced by |
|---|---|
| Execute an arbitrary command or script | Fixed `action` and `handler` enums; no command field in the scenario contract |
| Download or execute remote content | `network_access: none`; no HTTP client in execution handlers |
| Read arbitrary files or directories | Handler-specific paths; no generic file-read primitive |
| Access credentials, LSASS, SAM, browser stores, tokens, or keychains | No credential APIs or handlers; prohibited scenario class |
| Elevate privileges or impersonate another identity | Service identity is fixed; no token manipulation handlers |
| Move laterally or execute remotely | No SMB, WinRM, WMI, SSH, RDP, or remote-service handlers |
| Disable EDR, logging, firewall, or Sysmon | No service-control or policy-change handler for security controls |
| Clear event logs or delete evidence | Append-only local audit; no log-clearing handler |
| Encrypt, overwrite, or mass-delete data | No generic file mutation primitive; synthetic canary paths only |
| Create unbounded persistence | Only approved canary handlers with mandatory bounded cleanup |
| Exfiltrate raw event or customer data | Result schema permits minimum structured evidence only |
| Run without a signed, unexpired, host-bound job | Local signature, expiry, host, nonce, and replay validation |
| Continue after kill switch activation | Local scheduler gate and cooperative handler cancellation |

## Review rule

The absence of a capability is a product security property. Refactoring must not
introduce a generic primitive merely to reduce code duplication.
