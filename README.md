# StellaGate

StellaGate is a focused self-hosted VPS proxy console for people who want a
simple and dependable way to manage their own server.

It provides a streamlined daily-control experience built around a capable
Xray-based engine:

- clear VPS and service status;
- subscription links for common clients;
- one-click VLESS Reality and Hysteria2 switching;
- safe node restart, reset, and random-port actions;
- daily, monthly, and total traffic summaries;
- a clean interface that keeps advanced configuration out of the way.

## Quick install

Run as `root` on a supported Linux VPS:

```sh
curl -fsSL https://raw.githubusercontent.com/Ralph179/StellaGate-UI/codex/stellagate/install.sh | bash
```

After installation, the terminal prints the panel address, username, password,
and subscription link. No Cloud account, invitation code, or activation step is
required. Sign in to the panel to manage the node or switch protocols.

## Requirements

- A Linux VPS with root access
- A public IPv4 or IPv6 address
- `systemd` or another supported service manager
- Open firewall ports for the panel and selected proxy protocol

## Local development

```sh
# Backend
go run .

# Frontend development server
cd frontend
npm ci
npm run dev

# Production frontend build
npm run build
```

Open `/panel/` to access StellaGate.

## Stella API

All endpoints require the existing panel session or API bearer token.

| Method | Endpoint | Purpose |
| --- | --- | --- |
| GET | `/panel/api/stella/vps/status` | VPS, service, and protocol status |
| GET | `/panel/api/stella/subscription` | Subscription URL and QR payload |
| POST | `/panel/api/stella/subscription/reset` | Rotate the subscription token |
| POST | `/panel/api/stella/node/restart` | Restart the proxy service |
| POST | `/panel/api/stella/node/random-port` | Assign a random available port |
| POST | `/panel/api/stella/node/reset` | Perform a light, normal, or deep reset |
| POST | `/panel/api/stella/protocol/switch` | Switch between supported protocols |
| GET | `/panel/api/stella/traffic/summary` | Daily, monthly, and total traffic |

## Foundation and license

StellaGate is a modified derivative of
[MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui). This repository remains
licensed under **GPL-3.0-or-later**. Upstream copyright notices and license
terms are preserved.
