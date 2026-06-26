# StellaGate unattended-install contract

`install.sh` supports a non-interactive StellaGate bootstrap. The installer
writes the configured Cloud address to `/etc/x-ui/stellagate-cloud.json`.
Panel details are written to `/etc/x-ui/install-result.env` (mode `600`).

Full one-click install command:

```sh
curl -fsSL https://raw.githubusercontent.com/Ralph179/StellaGate-UI/codex/stellagate/install.sh | bash -s -- --cloud https://stellagate.simuse.uk
```

首次登录后按照面板提示操作即可。

协议不需要在安装时选择。激活后，StellaGate 面板里 VLESS Reality 和
Hysteria2 都可以直接切换使用。

| Flag | Accepted values | Current action |
| --- | --- | --- |
| `--panel` | `stellagate` | Enables post-install managed-node bootstrap. |
| `--cloud` | HTTPS Cloud URL | Official Cloud address used for invite activation. |

The same values can be supplied through `STELLAGATE_PANEL`,
and `STELLAGATE_CLOUD_URL`. Existing positional version installs remain
compatible.

The post-install stage authenticates locally with the one-time API token
created by the installer. Core StellaGate APIs stay locked until Cloud
activation succeeds. If no invite is supplied, the installer stops before node
creation and asks the user to activate in the panel. It fails clearly if the
panel does not start, activation fails, node creation fails, or the
subscription cannot be generated.

After activation, StellaGate-UI keeps checking Cloud validity. If Cloud revokes
or deletes the activation, the local panel is locked again, the managed
StellaGate node is disabled, and Xray is restarted. Network-only Cloud outages
are treated as temporary; explicit invalidation such as `revoked`,
`expired`, `invalid_token`, or device mismatch is enforced locally.

The original extension model remains deliberately narrow:

- StellaGate-UI only talks to an external Cloud activation API;
- activation tokens are stored only in `/etc/x-ui/stellagate-activation.json`
  with mode `600`;
- no VPS marketplace, payment, or multi-user control plane is introduced.
