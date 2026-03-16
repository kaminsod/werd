"""Mock captcha and email verification handlers.

These are no-op implementations that return immediately. In production,
they would integrate with captcha-solving services or email inbox polling.
Platforms may reject mock tokens — the create_account response will include
the specific error in that case.
"""

from playwright.async_api import Page


async def solve_captcha(page: Page, platform: str) -> str:
    """No-op captcha solver. Returns a mock token.

    In production, this would:
    - Detect the captcha type (reCAPTCHA, hCaptcha, Turnstile, etc.)
    - Call a solving service API or use platform-provided test keys
    - Return the verification token
    """
    # For platforms with test/sandbox modes, inject the test key here.
    return "mock-captcha-token"


async def verify_email(email: str, platform: str) -> str:
    """No-op email verification. Returns a mock verification URL.

    In production, this would:
    - Poll an email inbox (IMAP, API, or temp mail service)
    - Find the verification email from the platform
    - Extract and return the verification link
    """
    return f"https://{platform}.mock/verify?token=mock-verification-token"
