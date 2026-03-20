"""Bluesky browser automation via bsky.app."""

from __future__ import annotations

from playwright.async_api import Page

from .base import BasePlatform, ElementNotFoundError
from ..captcha import CaptchaService
from ..email import EmailVerifier
from ..models import CreateAccountResponse, PublishResponse, ReadResponse, ReadItem, ValidateResponse


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
        self,
        page: Page,
        email: str,
        username: str,
        password: str,
        captcha: CaptchaService | None = None,
        email_verifier: EmailVerifier | None = None,
    ) -> CreateAccountResponse:
        """Create a new Bluesky account via bsky.app.

        Bluesky signup flow:
        1. Navigate to bsky.app → "Create account"
        2. Select hosting provider (bsky.social)
        3. Enter email, password, date of birth
        4. Choose handle (username.bsky.social)
        5. CAPTCHA/verification (Cloudflare Turnstile in some flows)
        6. Account created
        """
        try:
            await page.goto(f"{self.BASE_URL}/")
            await page.wait_for_load_state("networkidle", timeout=15000)

            # Step 1: Click "Create account" — scoped to the landing dialog
            # to avoid clicking on elements hidden behind the overlay.
            dialog = page.locator('[role="dialog"]')
            if await dialog.count() > 0:
                create_btn = dialog.locator('text=Create account')
            else:
                create_btn = page.get_by_role("button", name="Create account")
            await create_btn.first.click(timeout=15000)
            await page.wait_for_timeout(3000)

            # Step 2: Hosting provider — usually defaults to bsky.social, click Next.
            next_btn = page.get_by_role("button", name="Next")
            try:
                await next_btn.click(timeout=5000)
                await page.wait_for_timeout(2000)
            except Exception:
                fallback = page.locator('[data-testid="nextBtn"]')
                if await fallback.count() > 0:
                    await fallback.click()
                    await page.wait_for_timeout(2000)

            # Step 3: Email + password + date of birth.
            await self._wait_and_fill(
                page,
                'input[type="email"]',
                email,
                "email input",
                timeout=10000,
            )

            await self._wait_and_fill(
                page,
                'input[type="password"]',
                password,
                "password input",
                timeout=5000,
            )

            # Date of birth — select a valid adult date.
            dob_input = page.locator(
                'input[type="date"], input[placeholder*="date"], input[placeholder*="birth"]'
            )
            if await dob_input.count() > 0:
                await dob_input.fill("1990-01-15")

            # Click Next to proceed to handle step.
            next_btn = page.get_by_role("button", name="Next")
            try:
                await next_btn.click(timeout=5000, force=True)
                await page.wait_for_timeout(2000)
            except Exception:
                pass

            # Step 4: Handle (username).
            handle_input = page.locator(
                'input[placeholder*="handle"], input[placeholder*="username"], '
                'input[placeholder*=".bsky.social"]'
            )
            if await handle_input.count() > 0:
                await handle_input.fill(username)
            else:
                # Fallback: look for the only text input on the page.
                text_inputs = page.locator('input[type="text"]')
                if await text_inputs.count() == 1:
                    await text_inputs.first.fill(username)

            # Step 5: Solve CAPTCHA if present (Bluesky uses Turnstile).
            if captcha:
                try:
                    has_turnstile = await page.locator(
                        'iframe[src*="turnstile"], .cf-turnstile'
                    ).count() > 0
                    if has_turnstile:
                        await captcha.solve(page, "turnstile")
                except Exception:
                    import logging
                    logging.getLogger(__name__).warning(
                        "Turnstile solve failed, continuing..."
                    )

            # Click Next / Create to finalize.
            for btn_name in ["Next", "Create Account", "Create", "Done"]:
                btn = page.get_by_role("button", name=btn_name)
                if await btn.count() > 0:
                    await btn.first.click(force=True)
                    await page.wait_for_timeout(3000)
                    break

            # Step 6: Skip profile setup if prompted.
            for skip_text in ["Skip", "Later", "Skip for now"]:
                skip_btn = page.locator(f'button:has-text("{skip_text}")')
                if await skip_btn.count() > 0:
                    await skip_btn.first.click()
                    await page.wait_for_load_state("networkidle", timeout=5000)
                    break

            # Step 7: Email verification if needed.
            if email_verifier:
                body_text = await page.locator("body").inner_text()
                if any(
                    phrase in body_text.lower()
                    for phrase in ["verify your email", "check your email", "confirmation"]
                ):
                    try:
                        verify_url = await email_verifier.wait_for_verification_link(
                            recipient=email,
                            sender_pattern="bluesky",
                            subject_pattern="verify",
                            timeout_secs=60,
                        )
                        await page.goto(verify_url)
                        await page.wait_for_load_state("networkidle", timeout=15000)
                    except Exception:
                        import logging
                        logging.getLogger(__name__).warning(
                            "Bluesky email verification failed"
                        )

            # Verify success.
            current_url = page.url
            body_text = await page.locator("body").inner_text()

            # Check for errors.
            error_indicators = [
                "handle is taken",
                "already in use",
                "invalid email",
                "too young",
                "something went wrong",
            ]
            body_lower = body_text.lower()
            for indicator in error_indicators:
                if indicator in body_lower:
                    return CreateAccountResponse(success=False, error=indicator)

            # Success: we reached the home feed.
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

            return CreateAccountResponse(
                success=False,
                error=f"signup may have been blocked by captcha or verification — URL: {current_url}",
            )
        except ElementNotFoundError as e:
            return CreateAccountResponse(success=False, error=str(e))
        except Exception as e:
            return CreateAccountResponse(success=False, error=str(e))
