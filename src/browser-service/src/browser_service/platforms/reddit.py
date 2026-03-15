"""Reddit browser automation via old.reddit.com."""

from playwright.async_api import Page

from .base import BasePlatform
from ..models import PublishResponse, ReadResponse, ReadItem, ValidateResponse


class RedditPlatform(BasePlatform):
    # Use old.reddit.com for simpler, more reliable automation.
    BASE_URL = "https://old.reddit.com"

    async def _login(self, page: Page, credentials: dict) -> None:
        username = credentials.get("username", "")
        password = credentials.get("password", "")
        if not username or not password:
            raise ValueError("reddit browser credentials require 'username' and 'password'")

        await page.goto(f"{self.BASE_URL}/login")
        await page.fill('input[name="user"]', username)
        await page.fill('input[name="passwd"]', password)
        await page.click('button[type="submit"]')
        await page.wait_for_url("**/", timeout=15000)

        # Verify login succeeded (check for username in nav).
        if await page.locator(".user").count() == 0:
            raise ValueError("reddit login failed — check credentials")

    async def validate(self, page: Page, credentials: dict) -> ValidateResponse:
        try:
            await self._login(page, credentials)
            return ValidateResponse(success=True)
        except Exception as e:
            return ValidateResponse(success=False, error=str(e))

    async def publish(self, page: Page, credentials: dict, content: str) -> PublishResponse:
        try:
            subreddit = credentials.get("subreddit", "")
            if not subreddit:
                return PublishResponse(success=False, error="subreddit is required in credentials")

            await self._login(page, credentials)

            # Navigate to submit page.
            await page.goto(f"{self.BASE_URL}/r/{subreddit}/submit?selftext=true")

            # Split content: first line = title, rest = body.
            lines = content.split("\n", 1)
            title = lines[0].strip()
            body = lines[1].strip() if len(lines) > 1 else ""

            if not title:
                title = "Post from Werd"

            # Fill the form.
            await page.fill('textarea[name="title"]', title)
            if body:
                await page.fill('textarea[name="text"]', body)

            # Submit.
            await page.click('button[name="submit"]')

            # Wait for redirect to the new post.
            await page.wait_for_url("**/comments/**", timeout=15000)

            return PublishResponse(
                success=True,
                post_id=page.url.split("/comments/")[1].split("/")[0] if "/comments/" in page.url else "",
                url=page.url,
            )
        except Exception as e:
            return PublishResponse(success=False, error=str(e))

    async def read(self, page: Page, credentials: dict, target: str) -> ReadResponse:
        try:
            subreddit = target or credentials.get("subreddit", "")
            if not subreddit:
                return ReadResponse(success=False, error="target subreddit is required")

            # No login needed for reading public subreddits.
            await page.goto(f"{self.BASE_URL}/r/{subreddit}/new")
            await page.wait_for_selector("#siteTable .thing", timeout=10000)

            items = []
            things = await page.locator("#siteTable .thing").all()
            for thing in things[:10]:
                title_el = thing.locator("a.title")
                title = await title_el.inner_text() if await title_el.count() > 0 else ""
                href = await title_el.get_attribute("href") if await title_el.count() > 0 else ""
                author_el = thing.locator("a.author")
                author = await author_el.inner_text() if await author_el.count() > 0 else ""
                time_el = thing.locator("time")
                date = await time_el.get_attribute("datetime") if await time_el.count() > 0 else ""

                url = href if href and href.startswith("http") else f"{self.BASE_URL}{href}"
                items.append(ReadItem(title=title, url=url, author=author, date=date))

            return ReadResponse(success=True, items=items)
        except Exception as e:
            return ReadResponse(success=False, error=str(e))
