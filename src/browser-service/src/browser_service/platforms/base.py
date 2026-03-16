"""Abstract base for platform automation."""

from abc import ABC, abstractmethod

from playwright.async_api import Page

from ..models import CreateAccountResponse, PublishResponse, ReadResponse, ValidateResponse


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
        self, page: Page, email: str, username: str, password: str
    ) -> CreateAccountResponse:
        ...

    async def _login(self, page: Page, credentials: dict) -> None:
        """Override in subclasses to implement platform-specific login."""
        raise NotImplementedError
