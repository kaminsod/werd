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
    """Generate a unique suffix for this test run."""
    import time, random, string
    ts = int(time.time())
    suffix = "".join(random.choices(string.ascii_lowercase, k=4))
    return f"{ts}_{suffix}"


# ── Account creation ──


@pytest.mark.timeout(60)
async def test_hn_create_account(hn, browser_page_factory, unique_id):
    """Create a real HN account with a unique username."""
    username = f"wt{unique_id}"  # HN usernames max ~15 chars
    password = f"TestPass_{unique_id}!"

    async with browser_page_factory() as page:
        result = await hn.create_account(page, "", username, password)

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
async def test_hn_validate_bad_password(hn, browser_page_factory):
    """Login with wrong password should fail."""
    if "username" not in _state:
        pytest.skip("Depends on test_hn_create_account")

    async with browser_page_factory() as page:
        result = await hn.validate(
            page, {"username": _state["username"], "password": "WrongPassword999!"}
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
    """Read /newest and verify our posted story appears."""
    if "post_title" not in _state:
        pytest.skip("Depends on test_hn_publish_text_post")

    async with browser_page_factory() as page:
        # Check the user's own submissions page for the post.
        result = await hn.read(
            page, {}, f"submitted?id={_state['username']}"
        )

    assert result.success, f"Read failed: {result.error}"

    titles = [item.title for item in result.items]
    found = any(_state["post_title"] in t for t in titles)
    assert found, (
        f"Could not find '{_state['post_title']}' in submitted items: {titles}"
    )
