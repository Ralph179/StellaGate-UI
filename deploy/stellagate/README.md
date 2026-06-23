# StellaGate unattended-install contract

`install.sh` supports a non-interactive StellaGate bootstrap. It installs the
compatible engine, waits for the local panel, creates exactly one managed
StellaGate inbound through the local Stella API, then writes the panel and
subscription URLs to `/etc/x-ui/install-result.env` (mode `600`).

```sh
curl -fsSL https://raw.githubusercontent.com/Ralph179/StellaGate-/codex/stellagate/install.sh | bash -s -- \
  --template vless-reality \
  --panel stellagate
```

`--token` is optional in the current local-only release. Its value is never
printed or stored; the result file records only whether one was supplied. This
keeps the CLI ready for a later cloud-registration service without making that
service a runtime dependency.

| Flag | Accepted values | Current action |
| --- | --- | --- |
| `--panel` | `stellagate` | Enables post-install managed-node bootstrap. |
| `--template` | `vless-reality`, `hysteria2` | Creates the selected protocol template. |
| `--token` | `SG_*` (optional) | Reserved for a future registration endpoint; not persisted. |

The same values can be supplied through `STELLAGATE_PANEL`,
`STELLAGATE_TEMPLATE`, and `STELLAGATE_INSTALL_TOKEN`. Existing positional
version installs remain compatible.

The post-install stage authenticates locally with the one-time API token
created by the installer. It fails clearly if the panel does not start, node
creation fails, or the subscription cannot be generated.

The original extension model remains deliberately narrow:

- a future registration endpoint may consume `--token` once, after install;
- no installation token enters the panel database;
- no VPS marketplace, payment, or multi-user control plane is introduced.
