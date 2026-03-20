"""Reddit browser automation.

Login and publish/read use old.reddit.com for simpler, more reliable automation.
Account creation uses new reddit (reddit.com) since old.reddit.com no longer
has a registration form.
"""

from __future__ import annotations

from playwright.async_api import Page

from .base import BasePlatform, ElementNotFoundError
from ..captcha import CaptchaService
from ..email import EmailVerifier
from ..models import CreateAccountResponse, PublishResponse, ReadResponse, ReadItem, ValidateResponse


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

    async def create_account(
        self,
        page: Page,
        email: str,
        username: str,
        password: str,
        captcha: CaptchaService | None = None,
        email_verifier: EmailVerifier | None = None,
    ) -> CreateAccountResponse:
        """Create a new Reddit account via reddit.com.

        Reddit's signup flow (2026):
        1. Enter email → Continue
        2. Verify email with 6-digit OTP code (sent to email)
        3. Enter username + password
        4. reCAPTCHA (if triggered)
        5. Account created
        """
        import logging
        log = logging.getLogger(__name__)

        try:
            await page.goto("https://www.reddit.com/account/register/")
            await page.wait_for_load_state("networkidle", timeout=15000)

            # --- Step 1: Email ---
            try:
                await self._wait_and_fill(
                    page,
                    'input[name="email"], input[type="email"], #regEmail',
                    email,
                    "email input on registration page",
                    timeout=15000,
                )
            except ElementNotFoundError:
                for selector in [
                    'a[href*="register"]',
                    'a:has-text("Sign Up")',
                    'button:has-text("Sign Up")',
                ]:
                    if await page.locator(selector).count() > 0:
                        await page.locator(selector).first.click()
                        await page.wait_for_load_state("networkidle", timeout=10000)
                        break
                await self._wait_and_fill(
                    page,
                    'input[name="email"], input[type="email"], #regEmail',
                    email,
                    "email input (after navigation)",
                    timeout=15000,
                )

            await self._wait_and_click(
                page,
                'button:has-text("Continue"), button:has-text("Next"), button[type="submit"]',
                "continue button after email",
                timeout=10000,
            )
            await page.wait_for_load_state("networkidle", timeout=10000)

            # --- Step 2: Email OTP verification ---
            # Reddit now requires a 6-digit code sent to the email before
            # proceeding to username/password. Wait for either the OTP input
            # or the username input to determine which step we're on.
            otp_or_username = page.locator(
                'input[placeholder*="erification"], input[placeholder*="code"], '
                'input[placeholder*="Code"], '
                'input[name="username"], #regUsername'
            )
            try:
                await otp_or_username.first.wait_for(timeout=15000)
            except Exception:
                pass

            body_text = await page.locator("body").inner_text()
            needs_otp = (
                "verify your email" in body_text.lower()
                or "verification code" in body_text.lower()
                or "digit code" in body_text.lower()
            )
            if needs_otp:
                if not email_verifier:
                    return CreateAccountResponse(
                        success=False,
                        error="Reddit requires email verification code but no email verifier configured",
                    )
                log.info("Reddit requesting email OTP code, polling Mailpit...")
                try:
                    code = await email_verifier.wait_for_verification_code(
                        recipient=email,
                        sender_pattern="reddit",
                        subject_pattern="",
                        timeout_secs=120,
                    )
                    log.info("Got verification code: %s", code)

                    # Fill the code input.
                    code_input = page.locator(
                        'input[type="text"], input[placeholder*="code"], '
                        'input[placeholder*="Code"], input[name*="code"]'
                    )
                    if await code_input.count() > 0:
                        await code_input.first.fill(code)
                    else:
                        return CreateAccountResponse(
                            success=False,
                            error="Found verification code but no code input field on page",
                        )

                    # Click Continue — try role-based locator first (handles
                    # React portals and non-standard button elements), then CSS.
                    clicked = False
                    for label in ["Continue", "Verify"]:
                        btn = page.get_by_role("button", name=label)
                        if await btn.count() > 0:
                            await btn.first.click()
                            clicked = True
                            break
                    if not clicked:
                        # Fallback to CSS selector.
                        await self._wait_and_click(
                            page,
                            'button:has-text("Continue"), button[type="submit"]',
                            "continue after OTP code",
                            timeout=10000,
                        )
                    await page.wait_for_load_state("networkidle", timeout=15000)
                except Exception as e:
                    return CreateAccountResponse(
                        success=False,
                        error=f"Email OTP verification failed: {e}",
                    )

            # --- Step 3: Username + Password ---
            # After email verification, Reddit shows username/password inputs.
            await self._wait_and_fill(
                page,
                'input[name="username"], #regUsername',
                username,
                "username input",
                timeout=15000,
            )

            await self._wait_and_fill(
                page,
                'input[name="password"], input[type="password"], #regPassword',
                password,
                "password input",
                timeout=10000,
            )

            # --- Step 4: Solve CAPTCHA ---
            # Reddit uses reCAPTCHA Enterprise (invisible, score-based).
            # Detect the type and solve accordingly.
            if captcha:
                try:
                    is_enterprise = await page.locator(
                        'iframe[src*="recaptcha/enterprise"], '
                        'script[src*="recaptcha/enterprise"]'
                    ).count() > 0
                    has_recaptcha = await page.locator(
                        'iframe[src*="recaptcha"], .g-recaptcha'
                    ).count() > 0

                    if is_enterprise:
                        log.info("Detected reCAPTCHA Enterprise, solving...")
                        await captcha.solve(page, "recaptcha_enterprise")
                    elif has_recaptcha:
                        await captcha.solve(page, "recaptcha_v2")
                except Exception as e:
                    log.warning("Captcha solve attempt: %s", e)

            # --- Step 5: Submit ---
            # Reddit's submit button text varies ("Continue", "Sign Up", etc.)
            # Use role-based locator for reliability.
            clicked = False
            for label in ["Continue", "Sign Up", "Create Account"]:
                btn = page.get_by_role("button", name=label)
                if await btn.count() > 0:
                    await btn.first.click()
                    clicked = True
                    break
            if not clicked:
                await self._wait_and_click(
                    page,
                    'button[type="submit"]',
                    "signup submit button",
                    timeout=10000,
                )
            await page.wait_for_load_state("networkidle", timeout=15000)

            # --- Verify success ---
            current_url = page.url
            body_text = await page.locator("body").inner_text()

            error_indicators = [
                "username is taken",
                "that username is already taken",
                "invalid email",
                "password must be",
                "something went wrong",
            ]
            body_lower = body_text.lower()
            for indicator in error_indicators:
                if indicator in body_lower:
                    return CreateAccountResponse(success=False, error=indicator)

            if "reddit.com" in current_url and "/register" not in current_url and "/account/register" not in current_url:
                return CreateAccountResponse(
                    success=True,
                    username=username,
                    credentials={
                        "username": username,
                        "password": password,
                    },
                )

            return CreateAccountResponse(
                success=False,
                error=f"signup may have been blocked — URL: {current_url}",
            )
        except ElementNotFoundError as e:
            return CreateAccountResponse(success=False, error=str(e))
        except Exception as e:
            return CreateAccountResponse(success=False, error=str(e))
