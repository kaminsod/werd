"""Basic API endpoint tests (no browser needed)."""

import httpx
import pytest


# ---------------------------------------------------------------------------
# Health check
# ---------------------------------------------------------------------------

async def test_healthz(api_client: httpx.AsyncClient):
    resp = await api_client.get("/healthz")
    assert resp.status_code == 200
    data = resp.json()
    assert data == {"status": "ok"}


# ---------------------------------------------------------------------------
# Unsupported-platform rejection tests
# ---------------------------------------------------------------------------

async def test_publish_unsupported_platform(api_client: httpx.AsyncClient):
    resp = await api_client.post(
        "/actions/publish",
        json={
            "platform": "twitter",
            "credentials": {},
            "content": "hello",
        },
    )
    data = resp.json()
    assert data["success"] is False
    assert "unsupported" in data["error"].lower()


async def test_validate_unsupported_platform(api_client: httpx.AsyncClient):
    resp = await api_client.post(
        "/actions/validate",
        json={
            "platform": "twitter",
            "credentials": {},
        },
    )
    data = resp.json()
    assert data["success"] is False


async def test_read_unsupported_platform(api_client: httpx.AsyncClient):
    resp = await api_client.post(
        "/actions/read",
        json={
            "platform": "twitter",
            "credentials": {},
        },
    )
    data = resp.json()
    assert data["success"] is False


async def test_create_account_unsupported_platform(api_client: httpx.AsyncClient):
    resp = await api_client.post(
        "/actions/create-account",
        json={
            "platform": "twitter",
            "username": "testuser",
            "password": "testpass",
        },
    )
    data = resp.json()
    assert data["success"] is False


# ---------------------------------------------------------------------------
# Mock helper tests (no browser, no network)
# ---------------------------------------------------------------------------

async def test_captcha_mock_returns_token():
    from browser_service.captcha import solve_captcha

    # solve_captcha expects a Page, but the mock ignores it.
    token = await solve_captcha(None, "reddit")  # type: ignore[arg-type]
    assert isinstance(token, str)
    assert len(token) > 0


async def test_verify_email_mock_returns_url():
    from browser_service.captcha import verify_email

    url = await verify_email("test@example.com", "reddit")
    assert isinstance(url, str)
    assert "reddit" in url
