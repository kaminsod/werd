"""Werd Browser Automation Service — FastAPI application."""

import base64
import traceback

from fastapi import FastAPI, Header, HTTPException

from .browser import browser_page
from .config import config
from .models import (
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

app = FastAPI(title="Werd Browser Service", version="0.1.0")


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

    try:
        async with browser_page(req.options) as page:
            result = await platform.create_account(page, req.email, req.username, req.password)

            # Screenshot on error.
            if not result.success and req.options.screenshot_on_error:
                try:
                    screenshot = await page.screenshot()
                    result.screenshot_b64 = base64.b64encode(screenshot).decode()
                except Exception:
                    pass

            return result
    except Exception as e:
        return CreateAccountResponse(success=False, error=f"browser error: {traceback.format_exc()}")
