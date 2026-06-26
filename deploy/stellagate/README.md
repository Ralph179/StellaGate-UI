# StellaGate unattended-install contract

`install.sh` supports a non-interactive StellaGate bootstrap. It installs the
compatible engine, waits for the local panel, creates exactly one managed
StellaGate inbound through the local Stella API, then writes the panel and
subscription URLs to `/etc/x-ui/install-result.env` (mode `600`).

```sh
curl -fsSL https://raw.githubusercontent.com/Ralph179/StellaGate-UI/codex/stellagate/install.sh | bash
```

默认是 StellaGate + VLESS Reality。高级选项才需要参数：

```sh
curl -fsSL https://raw.githubusercontent.com/Ralph179/StellaGate-UI/codex/stellagate/install.sh | bash -s -- --template hysteria2
```

如果要接入 StellaGate Cloud 邀请激活，只传 Cloud 地址即可。安装后首次
打开面板会显示激活页，用户输入邀请码后解锁本地面板：

```sh
curl -fsSL https://raw.githubusercontent.com/Ralph179/StellaGate-UI/codex/stellagate/install.sh | bash -s -- --cloud https://gate.example.com
```

如果安装时已经有邀请码，可以同时传入 `--invite`，安装脚本会调用本地
激活接口完成 claim，然后继续创建默认节点和订阅：

```sh
curl -fsSL https://raw.githubusercontent.com/Ralph179/StellaGate-UI/codex/stellagate/install.sh | bash -s -- --cloud https://gate.example.com --invite SGC-XXXX-XXXX-XXXX
```

| Flag | Accepted values | Current action |
| --- | --- | --- |
| `--panel` | `stellagate` | Enables post-install managed-node bootstrap. |
| `--template` | `vless-reality`, `hysteria2` | Creates the selected protocol template. |
| `--cloud` | HTTPS Cloud URL | Writes `/etc/x-ui/stellagate-cloud.json` and enables local activation lock. |
| `--invite` | `SGC-*` invite code | Claims activation through the configured Cloud; token is never printed. |

The same values can be supplied through `STELLAGATE_PANEL`,
`STELLAGATE_TEMPLATE`, `STELLAGATE_CLOUD_URL`, and
`STELLAGATE_INVITE_CODE`. Existing positional version installs remain
compatible.

The post-install stage authenticates locally with the one-time API token
created by the installer. In Cloud mode, core StellaGate APIs stay locked until
activation succeeds. It fails clearly if the panel does not start, activation
fails, node creation fails, or the subscription cannot be generated.

The original extension model remains deliberately narrow:

- StellaGate-UI only talks to an external Cloud activation API;
- activation tokens are stored only in `/etc/x-ui/stellagate-activation.json`
  with mode `600`;
- no VPS marketplace, payment, or multi-user control plane is introduced.
