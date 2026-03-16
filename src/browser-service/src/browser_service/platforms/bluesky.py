"""Bluesky browser automation via bsky.app."""

from playwright.async_api import Page

from .base import BasePlatform
from ..models import CreateAccountResponse, PublishResponse, ReadResponse, ReadItem, ValidateResponse
from ..captcha import solve_captcha, verify_email


class BlueskyPlatform(BasePlatform):
    BASE_URL = "https://bsky.app"

    async def _login(self, page: Page, credentials: dict) -> None:
        username = credentials.get("username", "")
        password = credentials.get("password", "")
        if not username or not password:
            raise ValueError("bluesky browser credentials require 'username' and 'password'")

        await page.goto(f"{self.BASE_URL}/")
        # Click "Sign in" if on the landing page.
        sign_in = page.get_by_role("link", name="Sign in")
        if await sign_in.is_visible():
            await sign_in.click()

        await page.fill('input[type="text"]', username)
        await page.fill('input[type="password"]', password)
        await page.get_by_role("button", name="Sign in").click()
        # Wait for navigation to home feed.
        await page.wait_for_url("**/", timeout=15000)

    async def validate(self, page: Page, credentials: dict) -> ValidateResponse:
        try:
            await self._login(page, credentials)
            return ValidateResponse(success=True)
        except Exception as e:
            return ValidateResponse(success=False, error=str(e))

    async def publish(self, page: Page, credentials: dict, content: str) -> PublishResponse:
        try:
            await self._login(page, credentials)

            # Open compose dialog.
            compose_btn = page.get_by_role("button", name="New post")
            if not await compose_btn.is_visible():
                # Try the floating action button.
                compose_btn = page.locator('[data-testid="composeFAB"]')
            await compose_btn.click()

            # Type content into the compose box.
            editor = page.locator('[data-testid="composePostView"] [contenteditable="true"]')
            await editor.fill(content)

            # Click publish.
            publish_btn = page.get_by_role("button", name="Post")
            await publish_btn.click()

            # Wait for the post to be created (dialog closes).
            await page.wait_for_timeout(3000)

            return PublishResponse(
                success=True,
                post_id="browser-post",
                url=page.url,
            )
        except Exception as e:
            return PublishResponse(success=False, error=str(e))

    async def read(self, page: Page, credentials: dict, target: str) -> ReadResponse:
        try:
            await self._login(page, credentials)

            # Navigate to target profile or feed.
            if target:
                await page.goto(f"{self.BASE_URL}/profile/{target}")
            # else stay on home feed.

            await page.wait_for_timeout(3000)

            # Scrape visible posts.
            items = []
            posts = page.locator('[data-testid="feedItem-by-"]').all()
            for post in await posts:
                title = await post.inner_text()
                items.append(ReadItem(
                    title=title[:200],
                    url=page.url,
                    content=title[:500],
                ))
                if len(items) >= 10:
                    break

            return ReadResponse(success=True, items=items)
        except Exception as e:
            return ReadResponse(success=False, error=str(e))

    async def create_account(
        self, page: Page, email: str, username: str, password: str
    ) -> CreateAccountResponse:
        """Create a new Bluesky account via bsky.app.

        Bluesky signup flow:
        1. Navigate to bsky.app → "Create account"
        2. Select hosting provider (bsky.social)
        3. Enter email, password, date of birth
        4. Choose handle (username.bsky.social)
        5. CAPTCHA/verification (mocked)
        6. Account created
        """
        try:
            await page.goto(f"{self.BASE_URL}/")
            await page.wait_for_timeout(2000)

            # Click "Create account" or "Create a new account".
            create_btn = page.get_by_role("link", name="Create account")
            if await create_btn.count() == 0:
                create_btn = page.locator('a:has-text("Create"), button:has-text("Create")')
            if await create_btn.count() > 0:
                await create_btn.first.click()
                await page.wait_for_timeout(2000)

            # Step 1: Hosting provider — usually defaults to bsky.social, click Next.
            next_btn = page.get_by_role("button", name="Next")
            if await next_btn.count() > 0:
                await next_btn.click()
                await page.wait_for_timeout(1000)

            # Step 2: Email + password + date of birth.
            email_input = page.locator('input[type="email"]')
            if await email_input.count() > 0:
                await email_input.fill(email)

            password_input = page.locator('input[type="password"]')
            if await password_input.count() > 0:
                await password_input.fill(password)

            # Date of birth — select a valid adult date.
            dob_input = page.locator('input[type="date"], input[placeholder*="date"], input[placeholder*="birth"]')
            if await dob_input.count() > 0:
                await dob_input.fill("1990-01-15")

            next_btn = page.get_by_role("button", name="Next")
            if await next_btn.count() > 0:
                await next_btn.click()
                await page.wait_for_timeout(2000)

            # Step 3: Handle (username).
            handle_input = page.locator('input[placeholder*="handle"], input[placeholder*="username"]')
            if await handle_input.count() > 0:
                await handle_input.fill(username)

            # Handle CAPTCHA (mock).
            await solve_captcha(page, "bluesky")

            next_btn = page.get_by_role("button", name="Next")
            if await next_btn.count() > 0:
                await next_btn.click()
                await page.wait_for_timeout(3000)

            # Step 4: Skip profile setup if prompted.
            skip_btn = page.locator('button:has-text("Skip"), button:has-text("Later")')
            if await skip_btn.count() > 0:
                await skip_btn.first.click()
                await page.wait_for_timeout(2000)

            # Handle email verification (mock).
            await verify_email(email, "bluesky")

            # Check if we reached the home feed (account created).
            current_url = page.url
            if "bsky.app" in current_url and "/login" not in current_url:
                handle = f"{username}.bsky.social"
                return CreateAccountResponse(
                    success=True,
                    username=handle,
                    credentials={
                        "identifier": handle,
                        "password": password,
                    },
                )

            # Check for errors.
            body_text = await page.locator("body").inner_text()
            error_indicators = [
                "handle is taken",
                "already in use",
                "invalid email",
                "too young",
            ]
            for indicator in error_indicators:
                if indicator in body_text.lower():
                    return CreateAccountResponse(success=False, error=indicator)

            return CreateAccountResponse(
                success=False,
                error=f"signup may have been blocked by captcha or verification — URL: {current_url}",
            )
        except Exception as e:
            return CreateAccountResponse(success=False, error=str(e))
