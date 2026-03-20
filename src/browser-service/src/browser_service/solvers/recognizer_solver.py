"""reCAPTCHA v2 solver using the recognizer library.

Recognizer uses ML models to solve reCAPTCHA v2 image challenges.
It requires Patchright (undetected Playwright fork) — standard Playwright
is fingerprinted and detected by reCAPTCHA.

Recognizer is GPL-3.0 licensed. It runs in-process here, but the browser
service is a separate Docker container, so there is no license infection
on the Apache-2.0 main codebase.
"""

from __future__ import annotations

import logging

from playwright.async_api import Page

from ..captcha import CaptchaSolver

logger = logging.getLogger(__name__)

_SUPPORTED_TYPES = {"recaptcha_v2"}


class RecognizerSolver(CaptchaSolver):
    """Solves reCAPTCHA v2 using recognizer's AsyncChallenger."""

    def supports(self, captcha_type: str) -> bool:
        return captcha_type in _SUPPORTED_TYPES

    async def solve(
        self, page: Page, captcha_type: str, site_key: str | None = None
    ) -> str:
        if captcha_type not in _SUPPORTED_TYPES:
            raise ValueError(f"RecognizerSolver does not support {captcha_type}")

        try:
            from recognizer.agents.playwright.async_control import AsyncChallenger
        except ImportError:
            try:
                from recognizer import AsyncChallenger
            except ImportError:
                raise RuntimeError(
                    "recognizer library not installed — "
                    "install with: pip install 'werd-browser-service[captcha]'"
                )

        logger.info("RecognizerSolver: attempting to solve reCAPTCHA v2")

        challenger = AsyncChallenger(page)
        # AsyncChallenger.solve() clicks the checkbox, handles image challenges,
        # and returns the g-recaptcha-response token.
        token = await challenger.solve()
        if not token:
            raise RuntimeError("RecognizerSolver: challenger returned empty token")

        logger.info("RecognizerSolver: successfully solved reCAPTCHA v2")
        return token
