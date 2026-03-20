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
# Captcha service tests
# ---------------------------------------------------------------------------

async def test_noop_solver_returns_token():
    from browser_service.captcha import NoopSolver

    solver = NoopSolver()
    assert solver.supports("recaptcha_v2")
    assert solver.supports("hcaptcha")
    assert solver.supports("turnstile")
    token = await solver.solve(None, "recaptcha_v2")  # type: ignore[arg-type]
    assert isinstance(token, str)
    assert len(token) > 0


async def test_captcha_service_priority():
    """CaptchaService tries solvers in order and returns first success."""
    from browser_service.captcha import CaptchaService, CaptchaSolver

    class FailingSolver(CaptchaSolver):
        async def solve(self, page, captcha_type, site_key=None):
            raise RuntimeError("intentional failure")
        def supports(self, captcha_type):
            return True

    class SucceedingSolver(CaptchaSolver):
        async def solve(self, page, captcha_type, site_key=None):
            return "success-token"
        def supports(self, captcha_type):
            return True

    service = CaptchaService([FailingSolver(), SucceedingSolver()])
    token = await service.solve(None, "recaptcha_v2")  # type: ignore[arg-type]
    assert token == "success-token"


async def test_captcha_service_all_fail():
    """CaptchaService raises when all solvers fail."""
    from browser_service.captcha import CaptchaService, CaptchaSolver, CaptchaSolveError

    class FailingSolver(CaptchaSolver):
        async def solve(self, page, captcha_type, site_key=None):
            raise RuntimeError("intentional failure")
        def supports(self, captcha_type):
            return True

    service = CaptchaService([FailingSolver()])
    with pytest.raises(CaptchaSolveError):
        await service.solve(None, "recaptcha_v2")  # type: ignore[arg-type]


async def test_captcha_service_skips_unsupported():
    """CaptchaService skips solvers that don't support the type."""
    from browser_service.captcha import CaptchaService, CaptchaSolver

    class V2OnlySolver(CaptchaSolver):
        async def solve(self, page, captcha_type, site_key=None):
            return "v2-token"
        def supports(self, captcha_type):
            return captcha_type == "recaptcha_v2"

    class AllSolver(CaptchaSolver):
        async def solve(self, page, captcha_type, site_key=None):
            return "all-token"
        def supports(self, captcha_type):
            return True

    service = CaptchaService([V2OnlySolver(), AllSolver()])

    # For v2, V2OnlySolver wins.
    token = await service.solve(None, "recaptcha_v2")  # type: ignore[arg-type]
    assert token == "v2-token"

    # For hcaptcha, V2OnlySolver is skipped, AllSolver wins.
    token = await service.solve(None, "hcaptcha")  # type: ignore[arg-type]
    assert token == "all-token"


# ---------------------------------------------------------------------------
# Email verifier tests
# ---------------------------------------------------------------------------

async def test_noop_verifier_returns_url():
    from browser_service.email import NoopVerifier

    verifier = NoopVerifier()
    url = await verifier.wait_for_verification_link("test@example.com")
    assert isinstance(url, str)
    assert "test@example.com" in url
