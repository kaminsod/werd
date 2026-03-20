"""Abstract base for platform automation."""

from __future__ import annotations

from abc import ABC, abstractmethod

from playwright.async_api import Page

from ..captcha import CaptchaService
from ..email import EmailVerifier
from ..models import CreateAccountResponse, PublishResponse, ReadResponse, ValidateResponse


class ElementNotFoundError(Exception):
    """Raised when a required page element is not found within timeout."""


class BasePlatform(ABC):
    @abstractmethod
    async def publish(self, page: Page, credentials: dict, content: str) -> PublishResponse:
        ...

    @abstractmethod
    async def validate(self, page: Page, credentials: dict) -> ValidateResponse:
        ...

    @abstractmethod
    async def read(self, page: Page, credentials: dict, target: str) -> ReadResponse:
        ...

    @abstractmethod
    async def create_account(
        self,
        page: Page,
        email: str,
        username: str,
        password: str,
        captcha: CaptchaService | None = None,
        email_verifier: EmailVerifier | None = None,
    ) -> CreateAccountResponse:
        ...

    async def _login(self, page: Page, credentials: dict) -> None:
        """Override in subclasses to implement platform-specific login."""
        raise NotImplementedError

    async def _wait_and_fill(
        self, page: Page, selector: str, value: str, description: str, timeout: int = 10000
    ) -> None:
        """Wait for an element to appear, then fill it. Raises on timeout."""
        try:
            await page.wait_for_selector(selector, timeout=timeout)
        except Exception:
            raise ElementNotFoundError(
                f"Element not found: {description} (selector: {selector})"
            )
        await page.fill(selector, value)

    async def _wait_and_click(
        self, page: Page, selector: str, description: str, timeout: int = 10000
    ) -> None:
        """Wait for an element to appear, then click it. Raises on timeout."""
        try:
            await page.wait_for_selector(selector, timeout=timeout)
        except Exception:
            raise ElementNotFoundError(
                f"Element not found: {description} (selector: {selector})"
            )
        await page.click(selector)
