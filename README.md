# StellaGate

StellaGate is a focused self-hosted VPS proxy console for people who own one
VPS and want a dependable connection without operating an "airport" platform.
It turns the capable 3x-ui/Xray engine into a deliberately small daily-control
experience:

- one VPS overview;
- subscription import for common clients;
- safe node reset levels;
- one-click VLESS Reality / Hysteria2 switching;
- clear traffic totals.

The default UI intentionally hides raw inbound, outbound, routing and JSON
configuration. Those capabilities remain in **Advanced settings** for recovery
and expert troubleshooting; they are not the product's primary workflow.

## One-click install

[Open the one-click installer](https://raw.githubusercontent.com/Ralph179/StellaGate-UI/codex/stellagate/install.sh)

```sh
curl -fsSL https://raw.githubusercontent.com/Ralph179/StellaGate-UI/codex/stellagate/install.sh | bash
```

The default install creates a StellaGate VLESS Reality node and subscription.
Use `--template hysteria2` only when you specifically need Hysteria2.

After installation, the terminal prints the panel URL, login username, login
password, and generated subscription URL. Open the panel URL, sign in, and use
the StellaGate home page to copy the subscription link or scan its QR code.

## Product boundaries

StellaGate is not a multi-tenant airport system. It does not provide payments,
plans, affiliate marketing, VPS purchasing, or an end-user mobile app. Version
one manages a single self-owned VPS.

## Run locally

```sh
# backend
go run .

# frontend development server
cd frontend
npm ci
npm run dev

# production frontend build (writes internal/web/dist)
npm run build
```

Open `/panel/` for StellaGate. `/panel/advanced` keeps the full engine UI for
expert administration.

## Stella API

All endpoints require the existing panel session or API bearer token.

| Method | Endpoint | Purpose |
| --- | --- | --- |
| GET | `/panel/api/stella/vps/status` | VPS, service and protocol status |
| GET | `/panel/api/stella/subscription` | Subscription URL and QR payload |
| POST | `/panel/api/stella/subscription/reset` | Rotate subscription token |
| POST | `/panel/api/stella/node/restart` | Restart proxy service |
| POST | `/panel/api/stella/node/random-port` | Assign a random available port |
| POST | `/panel/api/stella/node/reset` | `light`, `normal` or `deep` reset |
| POST | `/panel/api/stella/protocol/switch` | Switch to `vless-reality` or `hysteria2` |
| GET | `/panel/api/stella/traffic/summary` | Today, month and total traffic |

## Foundation and license

StellaGate is a modified derivative of
[MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui). The underlying advanced
engine is retained for compatibility and is not represented as StellaGate
product UX. This repository remains licensed under **GPL-3.0-or-later**; the
upstream copyright notices and license are preserved.

See [`deploy/stellagate/README.md`](deploy/stellagate/README.md) for installer
details and advanced bootstrap options.
