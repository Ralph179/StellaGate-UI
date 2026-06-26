# StellaGate unattended-install contract

`install.sh` supports a non-interactive StellaGate bootstrap. StellaGate-UI is
locked by default and requires a StellaGate-Cloud invitation before the local
panel can be used. The installer writes panel details to
`/etc/x-ui/install-result.env` (mode `600`).

```sh
curl -fsSL https://raw.githubusercontent.com/Ralph179/StellaGate-UI/codex/stellagate/install.sh | bash -s -- --cloud https://gate.example.com
```

只传 Cloud 地址时，安装后首次打开面板会显示激活页。用户输入
StellaGate-Cloud 邀请码后，本地面板才会解锁：

```sh
curl -fsSL https://raw.githubusercontent.com/Ralph179/StellaGate-UI/codex/stellagate/install.sh | bash -s -- --cloud https://gate.example.com
```

如果安装时已经有邀请码，可以同时传入 `--invite`，安装脚本会调用本地
激活接口完成 claim，然后继续创建默认节点和订阅：

```sh
curl -fsSL https://raw.githubusercontent.com/Ralph179/StellaGate-UI/codex/stellagate/install.sh | bash -s -- --cloud https://gate.example.com --invite SGC-XXXX-XXXX-XXXX
```

默认协议是 VLESS Reality。需要 Hysteria2 时才传 `--template`：

```sh
curl -fsSL https://raw.githubusercontent.com/Ralph179/StellaGate-UI/codex/stellagate/install.sh | bash -s -- --cloud https://gate.example.com --template hysteria2
```

| Flag | Accepted values | Current action |
| --- | --- | --- |
| `--panel` | `stellagate` | Enables post-install managed-node bootstrap. |
| `--template` | `vless-reality`, `hysteria2` | Creates the selected protocol template. |
| `--cloud` | HTTPS Cloud URL | Required. Writes `/etc/x-ui/stellagate-cloud.json` and enables local activation lock. |
| `--invite` | `SGC-*` invite code | Claims activation through the configured Cloud; token is never printed. |

The same values can be supplied through `STELLAGATE_PANEL`,
`STELLAGATE_TEMPLATE`, `STELLAGATE_CLOUD_URL`, and
`STELLAGATE_INVITE_CODE`. Existing positional version installs remain
compatible.

The post-install stage authenticates locally with the one-time API token
created by the installer. Core StellaGate APIs stay locked until Cloud
activation succeeds. If no invite is supplied, the installer stops before node
creation and asks the user to activate in the panel. It fails clearly if the
panel does not start, activation fails, node creation fails, or the
subscription cannot be generated.

The original extension model remains deliberately narrow:

- StellaGate-UI only talks to an external Cloud activation API;
- activation tokens are stored only in `/etc/x-ui/stellagate-activation.json`
  with mode `600`;
- no VPS marketplace, payment, or multi-user control plane is introduced.
