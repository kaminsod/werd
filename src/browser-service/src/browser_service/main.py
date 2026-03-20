"""Werd Browser Automation Service — FastAPI application."""

import base64
import logging
import traceback

from fastapi import FastAPI, Header, HTTPException

from .browser import browser_page
from .captcha import CaptchaService, NoopSolver
from .config import config
from .email import EmailVerifier, MailpitVerifier, NoopVerifier
from .models import (
    ActionOptions,
    CreateAccountRequest,
    CreateAccountResponse,
    HealthResponse,
    PublishRequest,
    PublishResponse,
    ReadRequest,
    ReadResponse,
    ValidateRequest,
    ValidateResponse,
)
from .platforms import get_platform

logger = logging.getLogger(__name__)

app = FastAPI(title="Werd Browser Service", version="0.1.0")


def _build_captcha_service() -> CaptchaService:
    """Build CaptchaService from config, trying solvers in priority order."""
    from .captcha import CaptchaSolver

    solver_names = [s.strip() for s in config.captcha_solvers.split(",") if s.strip()]
    solvers: list[CaptchaSolver] = []

    for name in solver_names:
        if name == "noop":
            solvers.append(NoopSolver())
        elif name == "recognizer":
            try:
                from .solvers.recognizer_solver import RecognizerSolver
                solvers.append(RecognizerSolver())
                logger.info("RecognizerSolver loaded")
            except ImportError:
                logger.warning(
                    "recognizer solver requested but not installed — skipping"
                )
        elif name == "api":
            if config.captcha_api_url and config.captcha_api_key:
                from .solvers.api_solver import ApiSolver
                solvers.append(ApiSolver(config.captcha_api_url, config.captcha_api_key))
                logger.info("ApiSolver loaded (url=%s)", config.captcha_api_url)
            else:
                logger.warning(
                    "api solver requested but CAPTCHA_API_URL/KEY not set — skipping"
                )
        else:
            logger.warning("Unknown captcha solver: %s — skipping", name)

    if not solvers:
        logger.warning("No captcha solvers configured, falling back to NoopSolver")
        solvers.append(NoopSolver())

    return CaptchaService(solvers)


def _build_email_verifier() -> EmailVerifier:
    """Build EmailVerifier from config."""
    if config.email_verifier == "mailpit":
        logger.info("Using MailpitVerifier (url=%s)", config.mailpit_url)
        return MailpitVerifier(config.mailpit_url)
    else:
        logger.info("Using NoopVerifier")
        return NoopVerifier()


# Build services at startup.
captcha_service = _build_captcha_service()
email_verifier = _build_email_verifier()


def _check_auth(x_internal_key: str | None):
    if config.internal_api_key and x_internal_key != config.internal_api_key:
        raise HTTPException(status_code=401, detail="invalid internal API key")


@app.get("/healthz", response_model=HealthResponse)
async def healthz():
    return HealthResponse(status="ok")


@app.post("/actions/publish", response_model=PublishResponse)
async def publish(req: PublishRequest, x_internal_key: str | None = Header(None)):
    _check_auth(x_internal_key)

    try:
        platform = get_platform(req.platform)
    except ValueError as e:
        return PublishResponse(success=False, error=str(e))

    try:
        async with browser_page(req.options) as page:
            result = await platform.publish(page, req.credentials, req.content)

            # Screenshot on error.
            if not result.success and req.options.screenshot_on_error:
                try:
                    screenshot = await page.screenshot()
                    result.screenshot_b64 = base64.b64encode(screenshot).decode()
                except Exception:
                    pass

            return result
    except Exception as e:
        return PublishResponse(success=False, error=f"browser error: {traceback.format_exc()}")


@app.post("/actions/read", response_model=ReadResponse)
async def read(req: ReadRequest, x_internal_key: str | None = Header(None)):
    _check_auth(x_internal_key)

    try:
        platform = get_platform(req.platform)
    except ValueError as e:
        return ReadResponse(success=False, error=str(e))

    try:
        async with browser_page(req.options) as page:
            return await platform.read(page, req.credentials, req.target)
    except Exception as e:
        return ReadResponse(success=False, error=f"browser error: {traceback.format_exc()}")


@app.post("/actions/validate", response_model=ValidateResponse)
async def validate(req: ValidateRequest, x_internal_key: str | None = Header(None)):
    _check_auth(x_internal_key)

    try:
        platform = get_platform(req.platform)
    except ValueError as e:
        return ValidateResponse(success=False, error=str(e))

    try:
        async with browser_page(req.options) as page:
            return await platform.validate(page, req.credentials)
    except Exception as e:
        return ValidateResponse(success=False, error=f"browser error: {traceback.format_exc()}")


@app.post("/actions/create-account", response_model=CreateAccountResponse)
async def create_account(req: CreateAccountRequest, x_internal_key: str | None = Header(None)):
    _check_auth(x_internal_key)

    try:
        platform = get_platform(req.platform)
    except ValueError as e:
        return CreateAccountResponse(success=False, error=str(e))

    # Use the longer account creation timeout.
    options = ActionOptions(
        timeout_secs=max(req.options.timeout_secs, config.account_timeout_secs),
        headless=req.options.headless,
        proxy=req.options.proxy,
        screenshot_on_error=req.options.screenshot_on_error,
    )

    try:
        async with browser_page(options) as page:
            result = await platform.create_account(
                page, req.email, req.username, req.password,
                captcha=captcha_service,
                email_verifier=email_verifier,
            )

            # Screenshot on error.
            if not result.success and options.screenshot_on_error:
                try:
                    screenshot = await page.screenshot()
                    result.screenshot_b64 = base64.b64encode(screenshot).decode()
                except Exception:
                    pass

            return result
    except Exception as e:
        return CreateAccountResponse(success=False, error=f"browser error: {traceback.format_exc()}")
