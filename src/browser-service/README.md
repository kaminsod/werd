# Browser Automation Service

Headless browser automation for platform interactions — account creation, credential validation, publishing, and reading. Built on [Patchright](https://github.com/Kaliiiiiiiiii-Vinyzu/patchright) (anti-detect Playwright fork) with pluggable captcha solving and email verification.

## Supported Platforms

| Platform | Login | Publish | Read | Account Creation | Captcha | Email Verification |
|----------|-------|---------|------|------------------|---------|-------------------|
| **Bluesky** | bsky.app | New post | Profile feed | Full E2E | Turnstile (auto-bypass via Patchright) | Not required |
| **Reddit** | old.reddit.com | Text/link/reply to r/subreddit | Subreddit feed | Email OTP + username/password (blocked by reCAPTCHA Enterprise) | reCAPTCHA Enterprise (requires paid API solver) | OTP code via Mailpit |
| **HN** | news.ycombinator.com | Text/link posts | /newest feed | Full E2E (when not IP rate-limited) | None | None |

## Architecture

```
FastAPI (:8091)
├── /actions/validate        → platform.validate(page, credentials)
├── /actions/publish         → platform.publish(page, credentials, content)
├── /actions/read            → platform.read(page, credentials, target)
└── /actions/create-account  → platform.create_account(page, email, username, password, captcha, email_verifier)

Captcha Service (priority chain):
├── RecognizerSolver  — ML-based, solves reCAPTCHA v2 image challenges (GPL-3.0, in-process)
└── ApiSolver         — Paid API (2Captcha/CapSolver), solves all types including reCAPTCHA Enterprise

Email Verification:
├── MailpitVerifier   — Polls Mailpit REST API for verification links or OTP codes
└── NoopVerifier      — Returns mock data (dev/test only)
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `INTERNAL_API_KEY` | — | Auth key for incoming requests |
| `BROWSER_HEADLESS` | `true` | Run Chromium headless |
| `BROWSER_TIMEOUT_SECS` | `30` | Default action timeout |
| `BROWSER_ACCOUNT_TIMEOUT_SECS` | `60` | Account creation timeout |
| `BROWSER_PROXY` | — | SOCKS proxy (e.g., `socks5://proxy:1080`) |
| `CAPTCHA_SOLVERS` | `noop` | Comma-separated solver list: `recognizer`, `api`, `noop` |
| `CAPTCHA_API_URL` | — | 2Captcha/CapSolver API endpoint |
| `CAPTCHA_API_KEY` | — | API key for paid captcha solver |
| `EMAIL_VERIFIER` | `noop` | Verifier backend: `mailpit` or `noop` |
| `MAILPIT_URL` | `http://mailpit:8025` | Mailpit REST API URL |
| `EMAIL_DOMAIN` | `verify.example.com` | Domain for verification email addresses |

## Captcha Solving

### RecognizerSolver (self-hosted, free)

Uses the [recognizer](https://github.com/Vinyzu/recognizer) library (GPL-3.0) to solve reCAPTCHA v2 image challenges via ML models. Runs in-process. Only supports `recaptcha_v2` — cannot solve reCAPTCHA v3, Enterprise, or Turnstile.

### ApiSolver (paid service)

Sends captcha tasks to a 2Captcha-compatible API. Supports all types:

| Type | Method | Notes |
|------|--------|-------|
| `recaptcha_v2` | `userrecaptcha` | Image challenge |
| `recaptcha_v3` | `userrecaptcha` + `version=v3` | Score-based (invisible) |
| `recaptcha_enterprise` | `userrecaptcha` + `enterprise=1` | Reddit uses this |
| `hcaptcha` | `hcaptcha` | |
| `turnstile` | `turnstile` | Cloudflare |

Cost: ~$2-3 per 1000 solves. Set `CAPTCHA_SOLVERS=recognizer,api` to try free solver first, fall back to paid.

## Platform-Specific Notes

### Reddit Account Creation

Reddit's 2026 signup flow: **Email → 6-digit OTP code → Username/Password → reCAPTCHA Enterprise**.

The OTP step works end-to-end (Mailpit receives the email, code is extracted from subject line). The final submit is blocked by reCAPTCHA Enterprise — an invisible, score-based system that requires a paid API solver. See [CAPTCHA_RESEARCH.md](../../design/CAPTCHA_RESEARCH.md) for details.

### Bluesky Account Creation

Works fully end-to-end. Turnstile CAPTCHA is auto-bypassed by Patchright. No paid services needed.

### Google Account Creation

**Not feasible via browser automation.** Google requires QR code verification from a real mobile device (triggered by TCP/IP fingerprinting at the OS level). See [CAPTCHA_RESEARCH.md](../../design/CAPTCHA_RESEARCH.md) for full analysis.

## Development

```bash
pip install -e ".[captcha]"           # Install with captcha solvers
patchright install --with-deps chromium  # Install browser

# Run tests
pytest tests/test_api.py -v           # Unit tests (no browser)
pytest tests/test_hn.py -v            # HN integration (real platform)
pytest tests/test_reddit.py -v        # Reddit integration (needs env vars)
pytest tests/test_bluesky.py -v       # Bluesky integration (needs env vars)
```

## Adding a New Platform

1. Create `src/browser_service/platforms/{name}.py` extending `BasePlatform`
2. Implement: `validate()`, `publish()`, `read()`, `create_account()`
3. Register in `platforms/__init__.py`: `"{name}": {Name}Platform`
4. Add credential schema in the frontend's `credential-editor.tsx`
5. Register adapter in Go API's `main.go`
