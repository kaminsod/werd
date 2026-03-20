"""Hacker News browser automation via news.ycombinator.com."""

from __future__ import annotations

from playwright.async_api import Page

from .base import BasePlatform
from ..captcha import CaptchaService
from ..email import EmailVerifier
from ..models import CreateAccountResponse, PublishResponse, ReadResponse, ReadItem, ValidateResponse


class HNPlatform(BasePlatform):
    BASE_URL = "https://news.ycombinator.com"

    async def _login(self, page: Page, credentials: dict) -> None:
        username = credentials.get("username", "")
        password = credentials.get("password", "")
        if not username or not password:
            raise ValueError("hn browser credentials require 'username' and 'password'")

        await page.goto(f"{self.BASE_URL}/login")
        # HN login form has two sets of inputs — use the login section.
        inputs = await page.locator('input[name="acct"]').all()
        await inputs[0].fill(username)
        pw_inputs = await page.locator('input[name="pw"]').all()
        await pw_inputs[0].fill(password)
        # Click the first submit button (login, not create account).
        submit_buttons = await page.locator('input[type="submit"]').all()
        await submit_buttons[0].click()

        # Wait for the logout link to appear (the reliable login success indicator).
        try:
            await page.wait_for_selector('a[href*="logout"]', timeout=15000)
        except Exception:
            raise ValueError("hn login failed — check credentials")

    async def validate(self, page: Page, credentials: dict) -> ValidateResponse:
        try:
            await self._login(page, credentials)
            return ValidateResponse(success=True)
        except Exception as e:
            return ValidateResponse(success=False, error=str(e))

    async def publish(self, page: Page, credentials: dict, content: str) -> PublishResponse:
        try:
            await self._login(page, credentials)

            # Navigate to submit page.
            await page.goto(f"{self.BASE_URL}/submit")

            # Split content: first line = title, rest = text (or URL).
            lines = content.split("\n", 1)
            title = lines[0].strip()
            body = lines[1].strip() if len(lines) > 1 else ""

            if not title:
                return PublishResponse(success=False, error="title is required (first line of content)")

            # Fill the form.
            await page.fill('input[name="title"]', title)

            # If body looks like a URL, put it in url field; otherwise use text.
            if body.startswith("http://") or body.startswith("https://"):
                await page.fill('input[name="url"]', body)
            elif body:
                await page.fill('textarea[name="text"]', body)

            # Submit.
            await page.click('input[type="submit"]')
            await page.wait_for_timeout(3000)

            # Check for success — should redirect to the new story or newest page.
            current_url = page.url
            post_id = ""
            if "item?id=" in current_url:
                post_id = current_url.split("item?id=")[1].split("&")[0]

            return PublishResponse(
                success=True,
                post_id=post_id,
                url=current_url,
            )
        except Exception as e:
            return PublishResponse(success=False, error=str(e))

    async def read(self, page: Page, credentials: dict, target: str) -> ReadResponse:
        try:
            # No login needed for reading.
            url = f"{self.BASE_URL}/newest"
            if target:
                url = f"{self.BASE_URL}/{target}"
            await page.goto(url)
            # Wait for stories to load — page may be empty for new users.
            try:
                await page.wait_for_selector(".athing", timeout=10000)
            except Exception:
                # Page loaded but has no items (e.g. new user with no submissions yet).
                return ReadResponse(success=True, items=[])

            items = []
            rows = await page.locator(".athing").all()
            for row in rows[:10]:
                title_el = row.locator(".titleline > a")
                title = await title_el.inner_text() if await title_el.count() > 0 else ""
                href = await title_el.get_attribute("href") if await title_el.count() > 0 else ""

                # Get the item ID for the HN URL.
                item_id = await row.get_attribute("id") or ""
                hn_url = f"{self.BASE_URL}/item?id={item_id}" if item_id else ""

                items.append(ReadItem(
                    title=title,
                    url=href if href and href.startswith("http") else hn_url,
                    content="",
                    author="",
                    date="",
                ))

            return ReadResponse(success=True, items=items)
        except Exception as e:
            return ReadResponse(success=False, error=str(e))

    async def create_account(
        self,
        page: Page,
        email: str,
        username: str,
        password: str,
        captcha: CaptchaService | None = None,
        email_verifier: EmailVerifier | None = None,
    ) -> CreateAccountResponse:
        """Create a new HN account.

        HN's login page has two forms:
        - Top: login (first set of acct/pw inputs + first submit)
        - Bottom: create account (second set of acct/pw inputs + second submit)

        HN has no captcha and no email verification.
        """
        try:
            await page.goto(f"{self.BASE_URL}/login")

            # Get the second set of inputs (create account section).
            acct_inputs = await page.locator('input[name="acct"]').all()
            pw_inputs = await page.locator('input[name="pw"]').all()
            submit_buttons = await page.locator('input[type="submit"]').all()

            if len(acct_inputs) < 2 or len(pw_inputs) < 2 or len(submit_buttons) < 2:
                return CreateAccountResponse(
                    success=False,
                    error="could not find create account form — page structure may have changed",
                )

            # Check if account creation is disabled (HN rate-limits by IP).
            await page.wait_for_timeout(1000)
            body_text = await page.locator("body").inner_text()
            if "account creation disabled" in body_text.lower() or "creation rate limit" in body_text.lower():
                return CreateAccountResponse(
                    success=False,
                    error="account creation disabled by HN (IP rate-limited)",
                )

            # Fill the create account form (second set of inputs).
            await acct_inputs[1].fill(username)
            await pw_inputs[1].fill(password)
            await submit_buttons[1].click()

            # Wait for either: navigation away from login page, or an error.
            # On success HN redirects to "/" and shows the user nav.
            # On failure it stays on /login with an error message.
            try:
                await page.wait_for_selector(
                    f'a[href="user?id={username}"], a[href*="logout"]',
                    timeout=15000,
                )
            except Exception:
                pass  # Fall through to error checking below.

            # Check for error messages first (before success checks).
            body_text = await page.locator("body").inner_text()
            body_lower = body_text.lower()
            error_indicators = [
                "that username is taken",
                "username is taken",
                "already exists",
                "account creation disabled",
                "creation rate limit",
            ]
            for indicator in error_indicators:
                if indicator in body_lower:
                    if "disabled" in indicator or "rate limit" in indicator:
                        return CreateAccountResponse(success=False, error="account creation disabled by HN (IP rate-limited)")
                    return CreateAccountResponse(success=False, error="username is already taken")

            # Check if account was created (user nav link appears).
            if await page.locator(f'a[href="user?id={username}"]').count() > 0:
                return CreateAccountResponse(
                    success=True,
                    username=username,
                    credentials={"username": username, "password": password},
                )

            # Check for logout link (logged in but user link selector didn't match exactly).
            if await page.locator('a[href*="logout"]').count() > 0:
                return CreateAccountResponse(
                    success=True,
                    username=username,
                    credentials={"username": username, "password": password},
                )

            return CreateAccountResponse(
                success=False,
                error=f"signup failed — page URL: {page.url}",
            )
        except Exception as e:
            return CreateAccountResponse(success=False, error=str(e))
