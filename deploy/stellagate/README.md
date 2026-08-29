# StellaGate unattended-install contract

`install.sh` supports a non-interactive StellaGate bootstrap. Panel details are
written to `/etc/x-ui/install-result.env` (mode `600`).

Full one-click install command:

```sh
curl -fsSL https://raw.githubusercontent.com/Ralph179/StellaGate-UI/codex/stellagate/install.sh | bash
```

首次登录后按照面板提示操作即可。

协议不需要在安装时选择。安装后，StellaGate 面板里 VLESS Reality 和
Hysteria2 都可以直接切换使用。

| Flag | Accepted values | Current action |
| --- | --- | --- |
| `--panel` | `stellagate` | Enables post-install managed-node bootstrap. |
| `--template` | `vless-reality`, `hysteria2` | Selects the initial managed node protocol. |

The same values can be supplied through `STELLAGATE_PANEL` and
`STELLAGATE_TEMPLATE`. Existing positional version installs remain compatible.

The post-install stage authenticates only to the local panel with the API token
created by the installer. It creates the initial node and subscription without
calling an external activation service. It fails clearly if the panel does not
start, node creation fails, or the subscription cannot be generated.

The product model remains deliberately narrow:

- StellaGate-UI can run fully independently on the user's VPS;
- no Cloud account, invitation code, or activation token is required;
- no VPS marketplace, payment, or multi-user control plane is introduced.
