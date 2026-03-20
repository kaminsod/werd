"""Reddit integration tests — real platform interactions.

These tests use a pre-provisioned Reddit account to publish and read
on real Reddit. Account creation uses the captcha/email services.

Required env vars (skip all if not set):
    WERD_TEST_REDDIT_USERNAME
    WERD_TEST_REDDIT_PASSWORD
    WERD_TEST_REDDIT_SUBREDDIT  (default: "test")

Run with:
    pytest tests/test_reddit.py -v --timeout=120
"""

import os

import pytest

from browser_service.platforms.reddit import RedditPlatform
from browser_service.captcha import CaptchaService, NoopSolver
from browser_service.email import NoopVerifier


# Skip entire module if credentials not provided.
_username = os.environ.get("WERD_TEST_REDDIT_USERNAME", "")
_password = os.environ.get("WERD_TEST_REDDIT_PASSWORD", "")
_subreddit = os.environ.get("WERD_TEST_REDDIT_SUBREDDIT", "test")

pytestmark = pytest.mark.skipif(
    not _username or not _password,
    reason="WERD_TEST_REDDIT_USERNAME / WERD_TEST_REDDIT_PASSWORD not set",
)

_state: dict = {}


@pytest.fixture(scope="module")
def reddit():
    return RedditPlatform()


@pytest.fixture(scope="module")
def credentials():
    return {
        "username": _username,
        "password": _password,
        "subreddit": _subreddit,
    }


@pytest.fixture(scope="module")
def unique_id():
    import time, random, string
    ts = int(time.time())
    suffix = "".join(random.choices(string.ascii_lowercase, k=4))
    return f"{ts}_{suffix}"


# ── Credential validation ──


@pytest.mark.timeout(60)
async def test_reddit_validate_credentials(reddit, browser_page_factory, credentials):
    """Login to real Reddit with provided credentials."""
    async with browser_page_factory() as page:
        result = await reddit.validate(page, credentials)

    assert result.success, f"Reddit login failed: {result.error}"


@pytest.mark.timeout(60)
async def test_reddit_validate_bad_password(reddit, browser_page_factory):
    """Login with wrong password should fail."""
    async with browser_page_factory() as page:
        result = await reddit.validate(
            page, {"username": _username, "password": "WrongPassword999!"}
        )

    assert not result.success


# ── Publishing ──


@pytest.mark.timeout(90)
async def test_reddit_publish_to_subreddit(
    reddit, browser_page_factory, credentials, unique_id
):
    """Publish a text post to r/test on real Reddit."""
    title = f"Werd Integration Test {unique_id}"
    body = "Automated integration test post from Werd. Please ignore."
    content = f"{title}\n{body}"

    async with browser_page_factory() as page:
        result = await reddit.publish(page, credentials, content)

    assert result.success, f"Reddit publish failed: {result.error}"
    assert result.url, "Expected a URL"
    assert "/comments/" in result.url, f"URL doesn't look like a post: {result.url}"

    _state["post_title"] = title
    _state["post_url"] = result.url
    _state["post_id"] = result.post_id


# ── Reading ──


@pytest.mark.timeout(60)
async def test_reddit_read_subreddit(reddit, browser_page_factory, credentials):
    """Read r/test/new — should return items."""
    async with browser_page_factory() as page:
        result = await reddit.read(page, credentials, _subreddit)

    assert result.success, f"Reddit read failed: {result.error}"
    assert len(result.items) > 0, f"Expected items in r/{_subreddit}"
    item = result.items[0]
    assert item.title, "Item should have a title"


@pytest.mark.timeout(60)
async def test_reddit_read_and_find_own_post(reddit, browser_page_factory, credentials):
    """Read r/test/new and find our published post."""
    if "post_title" not in _state:
        pytest.skip("Depends on test_reddit_publish_to_subreddit")

    async with browser_page_factory() as page:
        result = await reddit.read(page, credentials, _subreddit)

    assert result.success
    titles = [item.title for item in result.items]
    found = any(_state["post_title"] in t for t in titles)
    assert found, (
        f"Could not find '{_state['post_title']}' in r/{_subreddit}: {titles}"
    )


# ── Account creation ──


@pytest.mark.timeout(90)
async def test_reddit_create_account_with_noop(reddit, browser_page_factory, unique_id):
    """Attempt Reddit account creation with noop captcha/email.

    With NoopSolver, the captcha won't actually be solved, so this will
    likely fail with a captcha error — but it exercises the full flow.
    """
    username = f"werdtest{unique_id}"[:20]
    password = f"TestPass_{unique_id}!"

    noop_captcha = CaptchaService([NoopSolver()])
    noop_email = NoopVerifier()

    async with browser_page_factory() as page:
        result = await reddit.create_account(
            page, f"{username}@example.com", username, password,
            captcha=noop_captcha,
            email_verifier=noop_email,
        )

    # With noop solver, expect failure due to captcha.
    # The test validates the flow runs without crashing.
    assert isinstance(result.success, bool)
    if not result.success:
        assert result.error  # Should have a meaningful error message
