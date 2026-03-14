# Caddy Reverse Proxy

Caddy is the single entry point for all HTTP traffic. It handles TLS termination, subdomain routing, security headers, CORS, and WebSocket proxying.

## Caddyfile Variants

| File | Mode | TLS | Routing |
|---|---|---|---|
| `Caddyfile` | Production (cloud/residential) | Automatic Let's Encrypt (HTTP-01) | Subdomain-based (`werd.yourdomain.com`) |
| `Caddyfile.local` | Local-only | None | Port-based (`localhost:3080`) |

### Switching modes

In `docker-compose.yml`, swap the volume mount:

```yaml
# Production (default):
- ../caddy/Caddyfile:/etc/caddy/Caddyfile:ro

# Local-only:
- ../caddy/Caddyfile.local:/etc/caddy/Caddyfile:ro
```

Then restart Caddy: `podman-compose restart caddy`

## Subdomain Mapping (Production)

All subdomains use `{$WERD_DOMAIN}` — set `WERD_DOMAIN` in `.env`.

| Subdomain | Service | Internal Port |
|---|---|---|
| `werd.{domain}` | Werd Dashboard | 3000 |
| `api.{domain}` | Werd API | 8090 |
| `monitor.{domain}` | changedetection.io | 5000 |
| `rss.{domain}` | RSSHub | 1200 |
| `ntfy.{domain}` | ntfy | 2586 |
| `analytics.{domain}` | Umami | 3000 |

## Port Mapping (Local)

| Port | Service |
|---|---|
| 3080 | Werd Dashboard |
| 3081 | Werd API |
| 3082 | changedetection.io |
| 3083 | RSSHub |
| 3084 | ntfy |
| 3085 | Umami |

## Security Headers

Both Caddyfiles apply security headers via shared snippets:

| Header | Value | Purpose |
|---|---|---|
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains; preload` | Force HTTPS (production only) |
| `X-Content-Type-Options` | `nosniff` | Prevent MIME-type sniffing |
| `X-Frame-Options` | `DENY` | Prevent clickjacking |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Limit referrer leakage |
| `Permissions-Policy` | `camera=(), microphone=(), geolocation=()` | Disable unnecessary browser APIs |
| `-Server` | (removed) | Hide Caddy server header |

## CORS

The API subdomain (`api.{domain}`) includes CORS headers allowing requests from the dashboard subdomain (`werd.{domain}`). This is required because the SPA and API are on different subdomains.

In local mode, CORS is set to `*` since there's no domain to restrict to.

## WebSocket Support

Caddy proxies WebSocket connections for:
- **Werd API** — real-time alert feed (`/ws` endpoint)
- **ntfy** — push notification streaming

## Request Body Limits

| Service | Limit | Reason |
|---|---|---|
| Werd API | 10 MB | Webhook payloads, file uploads |
| Others | Caddy default (varies) | No special requirements |

## Adding a New Service

1. Add a route block in both `Caddyfile` and `Caddyfile.local`
2. Import the `security_headers` snippet
3. Add `request_body` limits if the service handles uploads
4. Uncomment the service in `docker-compose.yml`
5. Restart Caddy: `podman-compose restart caddy`
