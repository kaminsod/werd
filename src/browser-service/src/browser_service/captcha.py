"""Captcha solving framework with pluggable backends.

Solvers are tried in priority order (configured via CAPTCHA_SOLVERS env var).
The first solver that supports the captcha type and succeeds wins.
"""

from __future__ import annotations

import logging
from abc import ABC, abstractmethod

from playwright.async_api import Page

logger = logging.getLogger(__name__)


class CaptchaSolveError(Exception):
    """Raised when all solvers fail to solve a captcha."""


class CaptchaSolver(ABC):
    """Backend-agnostic captcha solver interface."""

    @abstractmethod
    async def solve(
        self, page: Page, captcha_type: str, site_key: str | None = None
    ) -> str:
        """Solve captcha on the given page. Returns token or raises."""
        ...

    @abstractmethod
    def supports(self, captcha_type: str) -> bool:
        """Whether this solver can handle the given captcha type."""
        ...


class NoopSolver(CaptchaSolver):
    """Returns a mock token. For dev/test only."""

    async def solve(
        self, page: Page, captcha_type: str, site_key: str | None = None
    ) -> str:
        logger.warning("NoopSolver: returning mock token for %s", captcha_type)
        return "mock-captcha-token"

    def supports(self, captcha_type: str) -> bool:
        return True


class CaptchaService:
    """Router that tries solvers in priority order."""

    def __init__(self, solvers: list[CaptchaSolver]) -> None:
        self.solvers = solvers

    async def solve(
        self, page: Page, captcha_type: str, site_key: str | None = None
    ) -> str:
        """Try each solver in order. Returns token from first success."""
        errors: list[str] = []
        for solver in self.solvers:
            if not solver.supports(captcha_type):
                continue
            try:
                token = await solver.solve(page, captcha_type, site_key)
                logger.info(
                    "Captcha solved by %s for type %s",
                    type(solver).__name__,
                    captcha_type,
                )
                return token
            except Exception as e:
                logger.warning(
                    "%s failed for %s: %s", type(solver).__name__, captcha_type, e
                )
                errors.append(f"{type(solver).__name__}: {e}")
        raise CaptchaSolveError(
            f"All solvers failed for {captcha_type}: {'; '.join(errors)}"
        )
