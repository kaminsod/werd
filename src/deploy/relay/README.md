# Relay VPS Setup

For residential deployments behind NAT/CGNAT. Run this on a small VPS ($3-5/mo, 512 MB RAM) to tunnel traffic to your home box.

## Architecture

```
[Internet] → [Relay VPS: Caddy + FRP Server] → [FRP Tunnel] → [Home Box: Werd Stack]
```

## Setup

1. Get a small VPS with a public IP
2. Point your domain's DNS to the VPS IP (wildcard A record: `*.yourdomain.com`)
3. Edit `frps.toml` — set a secure `auth.token`
4. Edit `Caddyfile` — replace `yourdomain.com` with your domain
5. `docker compose up -d`

On your home box, set `WERD_ACCESS_MODE=residential` in `.env` and configure the FRP client to connect to this VPS.
