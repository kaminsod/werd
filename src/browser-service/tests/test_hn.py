"""Hacker News integration tests — real platform interactions.

These tests create real accounts and posts on news.ycombinator.com.
No mocks. Requires network access and a working Playwright/Chromium install.

Run with:
    pytest tests/test_hn.py -v --timeout=120
"""

import pytest

from browser_service.models import ActionOptions
from browser_service.platforms.hn import HNPlatform


# Shared state across tests in this module (ordered test execution).
_state: dict = {}


@pytest.fixture(scope="module")
def hn():
    return HNPlatform()


@pytest.fixture(scope="module")
def options():
    return ActionOptions(headless=True, timeout_secs=30)


@pytest.fixture(scope="module")
def unique_id():
    """Generate a unique suffix for this test run.

    HN usernames are alphanumeric only, max ~15 chars.
    We use the last 6 digits of timestamp + 3 random alpha chars.
    """
    import time, random, string
    ts = str(int(time.time()))[-6:]
    suffix = "".join(random.choices(string.ascii_lowercase, k=3))
    return f"{ts}{suffix}"


# ── Account creation ──


@pytest.mark.timeout(60)
async def test_hn_create_account(hn, browser_page_factory, unique_id):
    """Create a real HN account with a unique username."""
    username = f"wt{unique_id}"  # e.g. "wt635133abc" — 11 chars, alphanumeric
    password = f"WerdTest{unique_id}x1"

    async with browser_page_factory() as page:
        result = await hn.create_account(page, "", username, password)

    if not result.success and "rate-limited" in result.error:
        pytest.skip(f"HN rate-limited account creation: {result.error}")

    assert result.success, f"Account creation failed: {result.error}"
    assert result.username == username
    assert result.credentials["username"] == username
    assert result.credentials["password"] == password

    # Store for subsequent tests.
    _state["username"] = username
    _state["password"] = password
    _state["credentials"] = {"username": username, "password": password}


@pytest.mark.timeout(60)
async def test_hn_create_account_duplicate(hn, browser_page_factory):
    """Attempting to create an account with an existing username should fail."""
    if "username" not in _state:
        pytest.skip("Depends on test_hn_create_account")

    async with browser_page_factory() as page:
        result = await hn.create_account(
            page, "", _state["username"], "AnyPassword123!"
        )

    assert not result.success
    assert "taken" in result.error.lower() or "unknown" in result.error.lower()


# ── Credential validation ──


@pytest.mark.timeout(60)
async def test_hn_validate_created_account(hn, browser_page_factory):
    """Validate that the created account can log in."""
    if "credentials" not in _state:
        pytest.skip("Depends on test_hn_create_account")

    async with browser_page_factory() as page:
        result = await hn.validate(page, _state["credentials"])

    assert result.success, f"Validation failed: {result.error}"


@pytest.mark.timeout(60)
async def test_hn_validate_bad_password(hn, browser_page_factory, unique_id):
    """Login with wrong password should fail."""
    # Use a known-good username format — doesn't need the created account.
    async with browser_page_factory() as page:
        result = await hn.validate(
            page, {"username": "dang", "password": "WrongPassword999!"}
        )

    assert not result.success


# ── Publishing ──


@pytest.mark.timeout(60)
async def test_hn_publish_text_post(hn, browser_page_factory, unique_id):
    """Publish a text post (Show HN style) to real HN."""
    if "credentials" not in _state:
        pytest.skip("Depends on test_hn_create_account")

    title = f"Werd Integration Test {unique_id}"
    body = "This is an automated integration test post. Please ignore."
    content = f"{title}\n{body}"

    async with browser_page_factory() as page:
        result = await hn.publish(page, _state["credentials"], content)

    assert result.success, f"Publish failed: {result.error}"
    assert result.url, "Expected a URL in the response"

    _state["post_title"] = title
    _state["post_url"] = result.url
    _state["post_id"] = result.post_id


@pytest.mark.timeout(60)
async def test_hn_publish_link_post(hn, browser_page_factory, unique_id):
    """Publish a link post to real HN."""
    if "credentials" not in _state:
        pytest.skip("Depends on test_hn_create_account")

    title = f"Werd Link Test {unique_id}"
    url = "https://example.com"
    content = f"{title}\n{url}"

    async with browser_page_factory() as page:
        result = await hn.publish(page, _state["credentials"], content)

    assert result.success, f"Link publish failed: {result.error}"
    assert result.url, "Expected a URL"


# ── Reading ──


@pytest.mark.timeout(60)
async def test_hn_read_newest(hn, browser_page_factory):
    """Read the newest stories page — should return items."""
    async with browser_page_factory() as page:
        result = await hn.read(page, {}, "newest")

    assert result.success, f"Read failed: {result.error}"
    assert len(result.items) > 0, "Expected at least one item on /newest"
    # Verify item structure.
    item = result.items[0]
    assert item.title, "Item should have a title"
    assert item.url, "Item should have a URL"


@pytest.mark.timeout(60)
async def test_hn_read_and_find_own_post(hn, browser_page_factory):
    """Verify our posted story is accessible on HN."""
    if "post_url" not in _state:
        pytest.skip("Depends on test_hn_publish_text_post")

    # Verify the post page itself is accessible (more reliable than scanning
    # the submissions list, which may not be indexed yet for new accounts).
    async with browser_page_factory() as page:
        await page.goto(_state["post_url"])
        await page.wait_for_timeout(2000)
        page_text = await page.locator("body").inner_text()

    assert _state["post_title"] in page_text, (
        f"Could not find '{_state['post_title']}' on post page {_state['post_url']}"
    )
