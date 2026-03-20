"""Browser lifecycle management."""

from contextlib import asynccontextmanager
from typing import AsyncGenerator

# Prefer Patchright (undetected Playwright fork) when available.
# Same API, but patches navigator.webdriver and other detection vectors.
try:
    from patchright.async_api import (
        Browser,
        BrowserContext,
        Page,
        async_playwright,
    )
except ImportError:
    from playwright.async_api import (  # type: ignore[assignment]
        Browser,
        BrowserContext,
        Page,
        async_playwright,
    )

from .models import ActionOptions
from .config import config


@asynccontextmanager
async def browser_page(options: ActionOptions) -> AsyncGenerator[Page, None]:
    """Create an ephemeral browser page with the given options.

    Launches a browser, creates a context (with optional proxy), yields a page,
    then cleans up everything. Each request gets a fresh browser context.
    """
    headless = options.headless if options.headless is not None else config.default_headless
    timeout_ms = (options.timeout_secs or config.default_timeout_secs) * 1000
    proxy_url = options.proxy or config.default_proxy

    pw = await async_playwright().start()
    try:
        browser: Browser = await pw.chromium.launch(headless=headless)
        try:
            context_kwargs: dict = {
                "user_agent": (
                    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 "
                    "(KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
                ),
                "viewport": {"width": 1920, "height": 1080},
                "locale": "en-US",
                "timezone_id": "America/New_York",
            }
            if proxy_url:
                context_kwargs["proxy"] = {"server": proxy_url}

            context: BrowserContext = await browser.new_context(**context_kwargs)
            context.set_default_timeout(timeout_ms)
            context.set_default_navigation_timeout(timeout_ms)

            page: Page = await context.new_page()
            try:
                yield page
            finally:
                await context.close()
        finally:
            await browser.close()
    finally:
        await pw.stop()
