"""Shared fixtures for browser-service tests."""

import random
import string
import time
from contextlib import asynccontextmanager

import httpx
import pytest

from browser_service.main import app
from browser_service.browser import browser_page
from browser_service.models import ActionOptions


@pytest.fixture
def test_timestamp() -> int:
    """Return current epoch timestamp for unique test data."""
    return int(time.time())


@pytest.fixture
def generate_username(test_timestamp: int):
    """Return a factory function that generates unique usernames."""

    def _generate(prefix: str = "werdtest") -> str:
        suffix = "".join(random.choices(string.ascii_lowercase + string.digits, k=4))
        return f"{prefix}_{test_timestamp}_{suffix}"

    return _generate


@pytest.fixture
async def api_client():
    """Async HTTP client wired to the FastAPI app (no network needed)."""
    transport = httpx.ASGITransport(app=app)
    async with httpx.AsyncClient(
        transport=transport,
        base_url="http://testserver",
        headers={"X-Internal-Key": ""},
    ) as client:
        yield client


@pytest.fixture
def browser_page_factory():
    """Return an async context manager that yields a real Playwright page.

    Usage in tests::

        async with browser_page_factory() as page:
            await page.goto("https://example.com")
    """

    @asynccontextmanager
    async def _factory(options: ActionOptions | None = None):
        opts = options or ActionOptions(headless=True)
        async with browser_page(opts) as page:
            yield page

    return _factory
