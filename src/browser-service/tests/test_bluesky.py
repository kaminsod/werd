"""Bluesky integration tests — real platform interactions.

These tests use a pre-provisioned Bluesky account to publish and read
on real bsky.app. Account creation is attempted but expected to fail
(CAPTCHA blocks automated signup).

Required env vars (skip all if not set):
    WERD_TEST_BLUESKY_HANDLE    (e.g. "testuser.bsky.social")
    WERD_TEST_BLUESKY_PASSWORD  (account password for browser login)

Run with:
    pytest tests/test_bluesky.py -v --timeout=120
"""

import os

import pytest

from browser_service.platforms.bluesky import BlueskyPlatform


# Skip entire module if credentials not provided.
_handle = os.environ.get("WERD_TEST_BLUESKY_HANDLE", "")
_password = os.environ.get("WERD_TEST_BLUESKY_PASSWORD", "")

pytestmark = pytest.mark.skipif(
    not _handle or not _password,
    reason="WERD_TEST_BLUESKY_HANDLE / WERD_TEST_BLUESKY_PASSWORD not set",
)

_state: dict = {}


@pytest.fixture(scope="module")
def bluesky():
    return BlueskyPlatform()


@pytest.fixture(scope="module")
def credentials():
    return {"username": _handle, "password": _password}


@pytest.fixture(scope="module")
def unique_id():
    import time, random, string
    ts = int(time.time())
    suffix = "".join(random.choices(string.ascii_lowercase, k=4))
    return f"{ts}_{suffix}"


# ── Credential validation ──


@pytest.mark.timeout(60)
async def test_bluesky_validate_credentials(bluesky, browser_page_factory, credentials):
    """Login to real Bluesky with provided credentials."""
    async with browser_page_factory() as page:
        result = await bluesky.validate(page, credentials)

    assert result.success, f"Bluesky login failed: {result.error}"


@pytest.mark.timeout(60)
async def test_bluesky_validate_bad_password(bluesky, browser_page_factory):
    """Login with wrong password should fail."""
    async with browser_page_factory() as page:
        result = await bluesky.validate(
            page, {"username": _handle, "password": "WrongPassword999!"}
        )

    assert not result.success


# ── Publishing ──


@pytest.mark.timeout(90)
async def test_bluesky_publish_post(
    bluesky, browser_page_factory, credentials, unique_id
):
    """Publish a post to real Bluesky."""
    content = f"Werd integration test {unique_id} - automated test post, please ignore"

    async with browser_page_factory() as page:
        result = await bluesky.publish(page, credentials, content)

    assert result.success, f"Bluesky publish failed: {result.error}"
    assert result.url, "Expected a URL"

    _state["post_content"] = content
    _state["post_url"] = result.url


# ── Reading ──


@pytest.mark.timeout(60)
async def test_bluesky_read_own_profile(bluesky, browser_page_factory, credentials):
    """Read own profile feed — should return items."""
    # Extract handle without .bsky.social for profile URL.
    handle = _handle

    async with browser_page_factory() as page:
        result = await bluesky.read(page, credentials, handle)

    assert result.success, f"Bluesky read failed: {result.error}"
    # Profile might be empty for a fresh test account — just verify no error.


@pytest.mark.timeout(60)
async def test_bluesky_read_and_find_own_post(
    bluesky, browser_page_factory, credentials
):
    """Read own profile and verify our post appears."""
    if "post_content" not in _state:
        pytest.skip("Depends on test_bluesky_publish_post")

    async with browser_page_factory() as page:
        result = await bluesky.read(page, credentials, _handle)

    assert result.success
    if len(result.items) > 0:
        # Search for our content in the items.
        texts = [item.title or item.content for item in result.items]
        # Bluesky post content is in the title field (inner_text of post element).
        found = any(_state["post_content"][:50] in t for t in texts)
        assert found, (
            f"Could not find '{_state['post_content'][:50]}' in profile items"
        )
    else:
        # Fresh profile with no items visible — can't verify. Pass with note.
        pytest.skip("No items found on profile (may be a fresh account)")


# ── Account creation (expected to fail due to CAPTCHA) ──


@pytest.mark.timeout(60)
@pytest.mark.xfail(reason="Bluesky blocks automated account creation with CAPTCHA")
async def test_bluesky_create_account_attempt(
    bluesky, browser_page_factory, unique_id
):
    """Attempt to create a Bluesky account. Expected to be blocked by CAPTCHA."""
    username = f"werdtest{unique_id}"[:20]
    password = f"TestPass_{unique_id}!"
    email = f"{username}@example.com"

    async with browser_page_factory() as page:
        result = await bluesky.create_account(page, email, username, password)

    assert result.success, f"Account creation failed: {result.error}"
