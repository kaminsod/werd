"""Generic paid captcha API solver (2Captcha/CapSolver/Anti-Captcha compatible).

These services accept a site key + page URL, solve the captcha remotely,
and return a token that can be injected into the page.
"""

from __future__ import annotations

import asyncio
import logging

from playwright.async_api import Page

from ..captcha import CaptchaSolver, CaptchaSolveError

logger = logging.getLogger(__name__)


class ApiSolver(CaptchaSolver):
    """Solves captchas via a 2Captcha-compatible HTTP API."""

    def __init__(self, api_url: str, api_key: str) -> None:
        self.api_url = api_url.rstrip("/")
        self.api_key = api_key

    def supports(self, captcha_type: str) -> bool:
        # Paid APIs handle all common captcha types.
        return True

    async def solve(
        self, page: Page, captcha_type: str, site_key: str | None = None
    ) -> str:
        try:
            import httpx
        except ImportError:
            raise RuntimeError(
                "httpx required for ApiSolver — install with: pip install httpx"
            )

        if not site_key:
            # Try to extract site key from the page.
            site_key = await self._extract_site_key(page, captcha_type)
            if not site_key:
                raise CaptchaSolveError(
                    f"No site_key provided and could not extract from page for {captcha_type}"
                )

        page_url = page.url
        logger.info(
            "ApiSolver: submitting %s task (site_key=%s, url=%s)",
            captcha_type,
            site_key[:8] + "...",
            page_url,
        )

        # Map captcha types to 2Captcha method names.
        method_map = {
            "recaptcha_v2": "userrecaptcha",
            "recaptcha_v3": "userrecaptcha",
            "recaptcha_enterprise": "userrecaptcha",
            "hcaptcha": "hcaptcha",
            "turnstile": "turnstile",
        }
        method = method_map.get(captcha_type, "userrecaptcha")

        async with httpx.AsyncClient(timeout=30) as client:
            # Step 1: Submit the captcha task.
            submit_params = {
                "key": self.api_key,
                "method": method,
                "googlekey": site_key,
                "pageurl": page_url,
                "json": 1,
            }
            if captcha_type == "recaptcha_v3":
                submit_params["version"] = "v3"
                submit_params["action"] = "verify"
                submit_params["min_score"] = "0.5"
            if captcha_type == "recaptcha_enterprise":
                submit_params["enterprise"] = 1

            resp = await client.post(f"{self.api_url}/in.php", data=submit_params)
            result = resp.json()
            if result.get("status") != 1:
                raise CaptchaSolveError(
                    f"API submit failed: {result.get('request', 'unknown error')}"
                )

            task_id = result["request"]
            logger.info("ApiSolver: task submitted, ID=%s", task_id)

            # Step 2: Poll for the result.
            poll_url = f"{self.api_url}/res.php"
            poll_params = {
                "key": self.api_key,
                "action": "get",
                "id": task_id,
                "json": 1,
            }

            for attempt in range(60):  # max ~2 minutes
                await asyncio.sleep(2)
                resp = await client.get(poll_url, params=poll_params)
                result = resp.json()

                if result.get("status") == 1:
                    token = result["request"]
                    logger.info("ApiSolver: solved after %d polls", attempt + 1)

                    # Inject the token into the page.
                    await self._inject_token(page, captcha_type, token)
                    return token

                if result.get("request") != "CAPCHA_NOT_READY":
                    raise CaptchaSolveError(
                        f"API solve failed: {result.get('request', 'unknown')}"
                    )

            raise CaptchaSolveError("API solve timed out after 120s")

    async def _extract_site_key(self, page: Page, captcha_type: str) -> str | None:
        """Try to extract the captcha site key from the page DOM."""
        selectors = {
            "recaptcha_v2": [
                ".g-recaptcha[data-sitekey]",
                "iframe[src*='recaptcha']",
            ],
            "recaptcha_v3": [
                ".g-recaptcha[data-sitekey]",
                "script[src*='recaptcha']",
            ],
            "recaptcha_enterprise": [
                ".g-recaptcha[data-sitekey]",
                "iframe[src*='recaptcha/enterprise']",
                "iframe[src*='recaptcha']",
                "script[src*='recaptcha/enterprise']",
            ],
            "hcaptcha": [
                ".h-captcha[data-sitekey]",
                "iframe[src*='hcaptcha']",
            ],
            "turnstile": [
                ".cf-turnstile[data-sitekey]",
                "iframe[src*='turnstile']",
            ],
        }

        for selector in selectors.get(captcha_type, []):
            el = page.locator(selector).first
            if await el.count() > 0:
                key = await el.get_attribute("data-sitekey")
                if key:
                    return key

                # Try extracting from iframe/script src.
                src = await el.get_attribute("src")
                if src:
                    if "k=" in src:
                        return src.split("k=")[1].split("&")[0]
                    if "render=" in src:
                        return src.split("render=")[1].split("&")[0]

        return None

    async def _inject_token(self, page: Page, captcha_type: str, token: str) -> None:
        """Inject the solved token into the page's captcha response field."""
        if captcha_type in ("recaptcha_v2", "recaptcha_v3", "recaptcha_enterprise"):
            await page.evaluate(
                """(token) => {
                    const el = document.getElementById('g-recaptcha-response');
                    if (el) { el.value = token; el.style.display = 'block'; }
                    // Also try textarea variant.
                    document.querySelectorAll('textarea[name="g-recaptcha-response"]')
                        .forEach(t => { t.value = token; });
                    // Trigger callback if registered.
                    if (typeof ___grecaptcha_cfg !== 'undefined') {
                        const clients = ___grecaptcha_cfg.clients;
                        if (clients) {
                            Object.values(clients).forEach(c => {
                                const cb = c?.['P']?.['P']?.callback || c?.callback;
                                if (typeof cb === 'function') cb(token);
                            });
                        }
                    }
                }""",
                token,
            )
        elif captcha_type == "hcaptcha":
            await page.evaluate(
                """(token) => {
                    const el = document.querySelector('textarea[name="h-captcha-response"]');
                    if (el) el.value = token;
                    const iframe = document.querySelector('iframe[src*="hcaptcha"]');
                    if (iframe) {
                        const id = iframe.getAttribute('data-hcaptcha-widget-id');
                        if (id && window.hcaptcha) window.hcaptcha.execute({id});
                    }
                }""",
                token,
            )
        elif captcha_type == "turnstile":
            await page.evaluate(
                """(token) => {
                    const el = document.querySelector('input[name="cf-turnstile-response"]');
                    if (el) el.value = token;
                    if (window.turnstile) {
                        const widgets = document.querySelectorAll('.cf-turnstile');
                        widgets.forEach(w => {
                            const cb = w.getAttribute('data-callback');
                            if (cb && typeof window[cb] === 'function') window[cb](token);
                        });
                    }
                }""",
                token,
            )
