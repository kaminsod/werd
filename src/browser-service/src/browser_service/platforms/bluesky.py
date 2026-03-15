"""Bluesky browser automation via bsky.app."""

from playwright.async_api import Page

from .base import BasePlatform
from ..models import PublishResponse, ReadResponse, ReadItem, ValidateResponse


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
