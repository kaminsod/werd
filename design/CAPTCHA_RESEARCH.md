# CAPTCHA & Account Creation Research

Comprehensive research findings on captcha solving and automated account creation across platforms. Last updated: 2026-03-20.

## Table of Contents

- [reCAPTCHA Types](#recaptcha-types)
- [Stealth Browser Benchmarks](#stealth-browser-benchmarks)
- [Self-Hosted Solver Research](#self-hosted-solver-research)
- [Reddit Account Creation](#reddit-account-creation)
- [Bluesky Account Creation](#bluesky-account-creation)
- [Google Account Creation](#google-account-creation)
- [SMS Verification Services](#sms-verification-services)
- [Recommendations](#recommendations)

---

## reCAPTCHA Types

| Type | Mechanism | Solvable Self-Hosted? | Notes |
|------|-----------|----------------------|-------|
| **v2 (checkbox + images)** | Image classification challenge | Yes — `recognizer` ML library | User clicks checkbox, may get image grid |
| **v3 (invisible)** | Score-based (0.0-1.0) | No | Score from Google's server-side model |
| **Enterprise** | Score-based + enterprise features | No | Same as v3, additional tuning options |
| **Turnstile** (Cloudflare) | Non-interactive JS challenge | Auto-bypassed by Patchright | Not Google, different mechanism |

### Why v3/Enterprise Can't Be Solved Locally

reCAPTCHA v3/Enterprise scoring depends on signals only Google has:

1. **Google account cookies** (SID, HSID) — logged-in Chrome users get 0.7-0.9 scores
2. **BotGuard VM** — obfuscated JavaScript VM that fingerprints the environment, runs timing checks, produces attestation tokens. Updated frequently by Google.
3. **Cross-site tracking** — aggregate browsing data across all Google properties
4. **IP reputation** — datacenter IPs penalized vs residential
5. **Behavioral signals** — mouse movement, scrolling, typing patterns

No local tool can fabricate Google account cookies or cross-site tracking data.

---

## Stealth Browser Benchmarks

From [techinz/browsers-benchmark](https://github.com/techinz/browsers-benchmark) (independent):

| Tool | Type | reCAPTCHA v3 Score | Anti-Bot Bypass Rate | Memory |
|------|------|-------------------|---------------------|--------|
| **Patchright** (our tool) | Chromium fork | **0.30** (best) | — | ~200MB |
| Camoufox | Firefox fork | 0.10 | 83.3% | ~1GB |
| NoDriver Chrome | Chrome CDP | 0.10 | 83.3% | — |
| Playwright Firefox | Standard Firefox | 0.10 | 83.3% | ~600MB |
| Standard Playwright | Chromium | 0.10 | 16.7% | ~200MB |
| **Human (no Google login)** | Any browser | **0.70** | — | — |
| **Human (Google logged in)** | Chrome | **0.90** | — | — |

**Key finding:** Best automation score (0.30) is well below the typical 0.50 pass threshold. The gap between 0.30 and 0.70 comes entirely from signals automation can't fake.

---

## Self-Hosted Solver Research

Exhaustive search for open-source, self-hostable reCAPTCHA v3/Enterprise solvers:

### Stealth Browsers

- **Patchright** (2.6k stars, Apache-2.0) — patches CDP `Runtime.Enable` leak, removes automation flags. Best v3 score (0.30), still fails.
- **Camoufox** (6.3k stars, Firefox) — C++ level fingerprint patching. Excellent for general anti-bot (83.3% bypass) but worst reCAPTCHA v3 score (0.10).
- **Botright** (952 stars, GPL-3.0) — claims 0.9 scores but contradicted by independent benchmarks.

### Token Harvesting

- **PyPasser** (572 stars, MIT, abandoned 2021) — sends HTTP requests to `/recaptcha/api/anchor` + `/reload`. Tokens lack BotGuard attestation → score ~0.1.
- **LOBYXLYX/recaptcha-v3-solver** (33 stars) — same approach, same limitation.

### BotGuard Reverse Engineering

- **0xBBFF/recaptcha-botguard** (22 stars) — disassembles bytecode, no token generation.
- **tomkabel/google-botguard-security-research** (79 stars) — documents VM architecture (register-based, self-modifying code, timing anti-debug).
- **LuanRT/BgUtils** (252 stars, MIT) — YouTube-specific PoTokens, not applicable to reCAPTCHA.

### Self-Hosted Solving Services

- **ohmycaptcha** (499 stars, MIT) — self-hosted captcha API using Playwright + Chromium. Supports `RecaptchaV3TaskProxyless` but tokens receive low scores (0.1-0.3) because the browser is still detected.

### Verdict

**No open-source, self-hosted solution achieves passing reCAPTCHA v3/Enterprise scores.** The score is computed server-side by Google using data only Google has. Paid API services (2Captcha, CapSolver, Anti-Captcha) at ~$2-3/1000 solves are the only reliable path.

---

## Reddit Account Creation

### Flow (2026)

```
1. Enter email → Click Continue
2. Receive 6-digit OTP code via email → Enter code → Click Continue
3. Enter username + password → Click Continue
4. reCAPTCHA Enterprise check (invisible, server-side)
5. Account created (or blocked by reCAPTCHA)
```

### Current Status

| Step | Status | Implementation |
|------|--------|---------------|
| Email entry | Working | Playwright fills form |
| OTP verification | Working | MailpitVerifier extracts code from subject line |
| Username/password | Working | Playwright fills form, role-based button clicks |
| reCAPTCHA Enterprise | **Blocked** | Requires paid API solver ($2-3/1000) |

### reCAPTCHA Details

- **Type:** reCAPTCHA Enterprise (invisible, score-based)
- **Site key:** `6LfirrMoAAAAAHZOipvza4kpp_VtTwLNuXVwURNQ`
- **Loaded via:** `recaptcha/enterprise.js?render=SITEKEY`
- **Error on failure:** "Something went wrong. Please try again."
- **Token field:** `g-recaptcha-response` textarea

### Technical Fixes Made

1. **STARTTLS on Mailpit** — Reddit's mail servers (via redditmail.com) refuse plain SMTP. Added self-signed TLS cert.
2. **OTP extraction from subject** — Reddit puts code in email subject: `"571599 is your Reddit verification code"`. Body text contains tracking IDs with false-positive 6-digit matches.
3. **React UI button handling** — Reddit's buttons require Playwright role-based locators (`get_by_role("button", name="Continue")`) not CSS selectors.
4. **Page transition timing** — `networkidle` fires before React renders OTP page. Fixed by waiting for input elements.

---

## Bluesky Account Creation

### Flow (2026)

```
1. Click "Create account" in landing dialog
2. Select hosting provider (bsky.social) → Next
3. Enter email + password + date of birth → Next
4. Choose handle (username.bsky.social) → Next
5. Turnstile CAPTCHA (auto-bypassed by Patchright)
6. Account created
```

### Current Status: **Fully Working**

All steps succeed end-to-end. Turnstile is auto-bypassed by Patchright with zero configuration — no paid services needed.

### Technical Fixes Made

1. **Landing dialog overlay** — bsky.app shows a modal dialog on load that intercepts all clicks. Fixed by scoping click to `page.locator('[role="dialog"]')`.
2. **Handle input selector** — placeholder is `.bsky.social`, not "handle" or "username". Added fallback selectors.

---

## Google Account Creation

### Why It's Not Feasible (2026)

Google account creation via browser automation is **blocked at the infrastructure level**, not just the browser level:

#### 1. QR Code Wall (TCP/IP Layer)

As of late 2025, Google uses **passive TCP/IP fingerprinting (p0f)** to classify connections at the network layer. Desktop OS fingerprints (Windows, Linux, macOS) are routed to **mandatory QR code verification** requiring a real Android/iOS device. SMS verification is only offered to mobile TCP fingerprints.

This decision happens *before* IP reputation is evaluated. Browser choice (Chrome, Firefox, Safari) is irrelevant — the block is at the OS TCP stack level.

#### 2. BotGuard

Google's obfuscated JavaScript VM runs environment checks including:
- Timing-based anti-debugger measures
- Self-modifying bytecode
- Browser API fingerprinting
- Attestation token generation

Updated frequently, making reverse engineering short-lived.

#### 3. Google Session Dependency

reCAPTCHA scores for Google properties depend heavily on being logged into a Google account — creating a circular dependency for account creation.

### What Would Be Required

The only reported working method (2026) combines:
1. **iPhone TCP fingerprint** (p0f spoofing at the network layer)
2. **macOS Safari browser fingerprint** (not Chrome, not Firefox)
3. **Dedicated 5G carrier proxy** (real carrier ASN, not datacenter)
4. **Non-VoIP SMS number** from a real carrier SIM
5. **72-hour warming period** post-creation

This is a fundamentally different class of problem from Reddit/Bluesky/HN automation.

### Firefox vs Chrome

Firefox does **not** help:
- QR code wall is browser-agnostic (TCP/IP layer)
- Camoufox (anti-detect Firefox) scores 0.10 on reCAPTCHA v3 (worst)
- Google account cookie integration only works in Chrome
- The only reported bypass uses macOS Safari, not Firefox

### Recommendation

Treat Google accounts as **manually provisioned**. Automate Gmail and Google Groups operations using existing accounts, not account creation.

---

## SMS Verification Services

### How They Work

Physical SIM farms with GSM gateways holding real carrier SIM cards. API flow:
1. `getNumber(country, service)` → rent a phone number
2. Enter number into platform's verification form
3. `waitSms(activationId, timeout)` → poll until SMS arrives
4. Extract code, complete verification
5. `setStatus(activationId, COMPLETED)` → release number

### Services (2026)

| Service | Status | Cost/verification | Notes |
|---------|--------|------------------|-------|
| SMS-Activate | **Shut down** (Dec 2025) | — | Largest service, closed abruptly |
| 5sim.net | Active | $0.01-0.50 | 358+ countries, Python library (`fivesim` on PyPI) |
| SMSPool | Active | $0.02-0.50 | Claims non-VoIP only |
| VoidMob | Active | Premium | Also sells 5G proxies with p0f spoofing |

### Reliability for Google

- **Pre-QR era:** ~60-80% success with non-VoIP numbers
- **Post-QR (2026):** SMS services alone don't work — QR code wall must be bypassed first
- Google limits ~2 accounts per phone number
- Accounts created with cheap virtual numbers get banned within hours to days

### Reliability for Non-Google Platforms

SMS services work well for Reddit, Bluesky, and other platforms that don't use TCP/IP fingerprinting or QR verification. Non-VoIP carrier numbers pass platform checks reliably.

### Risks

- **Legal:** SIM farms are illegal in most jurisdictions when used for fraud/spam
- **Account longevity:** Cheap numbers → fast bans; non-VoIP + warming → weeks to months
- **Service stability:** SMS-Activate (largest) shut down without warning in Dec 2025
- **Google ToS:** Automated account creation explicitly violates Terms of Service

---

## Recommendations

### Per-Platform Strategy

| Platform | Account Creation | Publishing/Reading |
|----------|-----------------|-------------------|
| **HN** | Browser automation (no captcha) | Browser automation |
| **Bluesky** | Browser automation (Turnstile auto-bypass) | API (AT Protocol) or browser |
| **Reddit** | Manual or paid API solver ($2-3/1000) | API (OAuth) or browser |
| **Google** | **Manual only** | Gmail API (OAuth) or browser automation |

### Captcha Configuration

```env
# HN + Bluesky (free, self-hosted):
CAPTCHA_SOLVERS=recognizer

# Add Reddit account creation (paid):
CAPTCHA_SOLVERS=recognizer,api
CAPTCHA_API_URL=https://2captcha.com
CAPTCHA_API_KEY=your-key
```

### Cost Estimates

| Operation | Cost | Frequency |
|-----------|------|-----------|
| Bluesky account creation | Free | As needed |
| HN account creation | Free | As needed (IP rate limit applies) |
| Reddit account creation | ~$0.003/account (captcha) | As needed |
| Google account creation | N/A (manual) | — |
