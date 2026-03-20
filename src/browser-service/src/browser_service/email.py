"""Email verification framework with pluggable backends.

Used during account creation flows to retrieve verification links
or OTP codes from confirmation emails sent by platforms.
"""

from __future__ import annotations

import logging
import re
from abc import ABC, abstractmethod

logger = logging.getLogger(__name__)

# Pattern to extract URLs from email HTML/text bodies.
_URL_RE = re.compile(r'https?://[^\s<>"\']+')

# Pattern to extract standalone 5-8 digit OTP codes.
_OTP_RE = re.compile(r'\b(\d{5,8})\b')


class EmailVerifyError(Exception):
    """Raised when email verification fails."""


class EmailVerifier(ABC):
    """Backend-agnostic email verification."""

    @abstractmethod
    async def wait_for_verification_link(
        self,
        recipient: str,
        sender_pattern: str = "",
        subject_pattern: str = "",
        timeout_secs: int = 60,
    ) -> str:
        """Poll for a verification email. Returns the link URL."""
        ...

    async def wait_for_verification_code(
        self,
        recipient: str,
        sender_pattern: str = "",
        subject_pattern: str = "",
        timeout_secs: int = 60,
    ) -> str:
        """Poll for a verification email. Returns the OTP code."""
        raise NotImplementedError("This verifier does not support OTP codes")


class NoopVerifier(EmailVerifier):
    """Returns immediately with a fake URL/code. For dev/test only."""

    async def wait_for_verification_link(
        self,
        recipient: str,
        sender_pattern: str = "",
        subject_pattern: str = "",
        timeout_secs: int = 60,
    ) -> str:
        logger.warning("NoopVerifier: returning mock URL for %s", recipient)
        return f"https://noop.mock/verify?to={recipient}"

    async def wait_for_verification_code(
        self,
        recipient: str,
        sender_pattern: str = "",
        subject_pattern: str = "",
        timeout_secs: int = 60,
    ) -> str:
        logger.warning("NoopVerifier: returning mock code for %s", recipient)
        return "123456"


class MailpitVerifier(EmailVerifier):
    """Polls Mailpit REST API for verification emails.

    Mailpit API docs: https://mailpit.axllent.org/docs/api-v1/
    """

    def __init__(self, mailpit_url: str) -> None:
        self.mailpit_url = mailpit_url.rstrip("/")

    async def wait_for_verification_link(
        self,
        recipient: str,
        sender_pattern: str = "",
        subject_pattern: str = "",
        timeout_secs: int = 60,
    ) -> str:
        import asyncio

        try:
            import httpx
        except ImportError:
            raise EmailVerifyError(
                "httpx is required for MailpitVerifier — install with: pip install httpx"
            )

        deadline = asyncio.get_event_loop().time() + timeout_secs
        poll_interval = 3  # seconds

        async with httpx.AsyncClient(timeout=10) as client:
            while asyncio.get_event_loop().time() < deadline:
                # Search for messages to this recipient.
                search_query = f"to:{recipient}"
                if subject_pattern:
                    search_query += f" subject:{subject_pattern}"

                resp = await client.get(
                    f"{self.mailpit_url}/api/v1/search",
                    params={"query": search_query},
                )
                if resp.status_code != 200:
                    logger.warning(
                        "Mailpit search returned %d: %s", resp.status_code, resp.text
                    )
                    await asyncio.sleep(poll_interval)
                    continue

                data = resp.json()
                messages = data.get("messages", [])
                if not messages:
                    logger.debug("No messages yet for %s, polling...", recipient)
                    await asyncio.sleep(poll_interval)
                    continue

                # Get the most recent message.
                msg_id = messages[0]["ID"]
                msg_resp = await client.get(
                    f"{self.mailpit_url}/api/v1/message/{msg_id}"
                )
                if msg_resp.status_code != 200:
                    await asyncio.sleep(poll_interval)
                    continue

                msg_data = msg_resp.json()

                # Extract verification link from HTML body, fall back to text.
                body = msg_data.get("HTML", "") or msg_data.get("Text", "")
                urls = _URL_RE.findall(body)

                # Filter for likely verification URLs.
                verify_keywords = [
                    "verify", "confirm", "activate", "validate", "token", "click",
                ]
                for url in urls:
                    url_lower = url.lower()
                    if any(kw in url_lower for kw in verify_keywords):
                        logger.info("Found verification link: %s", url)
                        return url

                # If no verification-specific URL, return the first URL.
                if urls:
                    logger.info(
                        "No verification-specific URL found, using first: %s", urls[0]
                    )
                    return urls[0]

                logger.debug(
                    "Message found but no URLs extracted for %s", recipient
                )
                await asyncio.sleep(poll_interval)

        raise EmailVerifyError(
            f"No verification email found for {recipient} within {timeout_secs}s"
        )

    async def wait_for_verification_code(
        self,
        recipient: str,
        sender_pattern: str = "",
        subject_pattern: str = "",
        timeout_secs: int = 60,
    ) -> str:
        import asyncio

        try:
            import httpx
        except ImportError:
            raise EmailVerifyError(
                "httpx is required for MailpitVerifier — install with: pip install httpx"
            )

        deadline = asyncio.get_event_loop().time() + timeout_secs
        poll_interval = 3

        async with httpx.AsyncClient(timeout=10) as client:
            while asyncio.get_event_loop().time() < deadline:
                search_query = f"to:{recipient}"
                if subject_pattern:
                    search_query += f" subject:{subject_pattern}"

                resp = await client.get(
                    f"{self.mailpit_url}/api/v1/search",
                    params={"query": search_query},
                )
                if resp.status_code != 200:
                    await asyncio.sleep(poll_interval)
                    continue

                data = resp.json()
                messages = data.get("messages", [])
                if not messages:
                    logger.debug("No messages yet for %s, polling...", recipient)
                    await asyncio.sleep(poll_interval)
                    continue

                msg_id = messages[0]["ID"]
                msg_resp = await client.get(
                    f"{self.mailpit_url}/api/v1/message/{msg_id}"
                )
                if msg_resp.status_code != 200:
                    await asyncio.sleep(poll_interval)
                    continue

                msg_data = msg_resp.json()

                # Try subject line first (many platforms put the code there).
                subject = msg_data.get("Subject", "")
                subject_codes = _OTP_RE.findall(subject)
                for code in subject_codes:
                    if 5 <= len(code) <= 8:
                        logger.info("Found verification code in subject: %s", code)
                        return code

                # Fall back to text body, stripping URLs first to avoid
                # matching tracking IDs embedded in links.
                body = msg_data.get("Text", "") or msg_data.get("HTML", "")
                body_no_urls = _URL_RE.sub("", body)
                codes = _OTP_RE.findall(body_no_urls)
                if codes:
                    for code in codes:
                        if 5 <= len(code) <= 8:
                            logger.info("Found verification code in body: %s", code)
                            return code

                logger.debug(
                    "Message found but no OTP code extracted for %s", recipient
                )
                await asyncio.sleep(poll_interval)

        raise EmailVerifyError(
            f"No verification code found for {recipient} within {timeout_secs}s"
        )
