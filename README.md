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
configuration so the product stays focused on the single-VPS workflow.

## One-click install

Full one-click install command:

```sh
curl -fsSL https://raw.githubusercontent.com/Ralph179/StellaGate-UI/codex/stellagate/install.sh | bash -s -- --cloud https://stellagate.888.cab
```

After installation, the terminal prints the panel URL, login username and login
password. Follow the panel prompts after the first login. VLESS Reality and
Hysteria2 are both available from the StellaGate home page; use protocol
switching in the panel instead of choosing a protocol during installation.
Then copy the subscription link or scan its QR code.

If the activation is revoked or deleted in StellaGate-Cloud, StellaGate-UI
re-checks Cloud status, relocks the local panel, disables the managed node, and
restarts the proxy service. Temporary Cloud network outages are tolerated, but
definitive Cloud invalidation is not cached as active.

## Product boundaries

StellaGate is not a multi-tenant airport system. It does not provide payments,
plans, affiliate marketing, VPS purchasing, or an end-user mobile app. Version
one manages a single self-owned VPS. StellaGate-UI is not usable as a standalone
uninvited panel; it must be unlocked by StellaGate-Cloud activation.

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

Open `/panel/` for StellaGate.

## Stella API

All endpoints require the existing panel session or API bearer token.

| Method | Endpoint | Purpose |
| --- | --- | --- |
| GET | `/panel/api/stella/activation/status` | Local activation status |
| POST | `/panel/api/stella/activation/claim` | Claim an invite code through configured Cloud |
| POST | `/panel/api/stella/activation/check` | Check activation validity and relock if revoked |
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
engine is kept behind StellaGate's simplified product layer and is not
represented as StellaGate product UX. This repository remains licensed under **GPL-3.0-or-later**; the
upstream copyright notices and license are preserved.

See [`deploy/stellagate/README.md`](deploy/stellagate/README.md) for installer
details.
