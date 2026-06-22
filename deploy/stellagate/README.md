# StellaGate unattended-install contract

`install.sh` already supports safe non-interactive installation through
environment variables and writes credentials to `/etc/x-ui/install-result.env`.
The StellaGate registration step should be added as a small post-install stage,
after the panel service has started and before the result file is printed.

Planned invocation:

```sh
curl -fsSL https://setup.example.com/install.sh | bash -s -- \
  --token SG_xxxxx \
  --template vless-reality \
  --panel stellagate
```

The future argument adapter maps these flags to environment variables so the
existing `install.sh` remains backward compatible:

| Flag | Environment variable | First-version action |
| --- | --- | --- |
| `--token` | `STELLAGATE_INSTALL_TOKEN` | Send only to the registration endpoint; never log it. |
| `--template` | `STELLAGATE_TEMPLATE` | Call `POST /panel/api/stella/protocol/switch` with `vless-reality` or `hysteria2`. |
| `--panel stellagate` | `STELLAGATE_PANEL` | Enables the post-install StellaGate bootstrap. |

The post-install stage should authenticate locally using the credentials in
`install-result.env`, create the default managed inbound through the Stella API,
then print only the panel URL and login username.  It must fail closed on an
unknown template, a registration error, or a failed proxy restart.  Token
handling stays separate from the panel database, leaving room for a later
cloud-registration service without turning this project into a VPS marketplace.
