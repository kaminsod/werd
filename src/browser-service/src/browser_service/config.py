import os


class Config:
    internal_api_key: str = os.getenv("INTERNAL_API_KEY", "")
    default_headless: bool = os.getenv("BROWSER_HEADLESS", "true").lower() == "true"
    default_timeout_secs: int = int(os.getenv("BROWSER_TIMEOUT_SECS", "30"))
    default_proxy: str = os.getenv("BROWSER_PROXY", "")

    # Account creation gets a longer timeout (registration flows are slow).
    account_timeout_secs: int = int(os.getenv("BROWSER_ACCOUNT_TIMEOUT_SECS", "60"))

    # Captcha solver config.
    captcha_solvers: str = os.getenv("CAPTCHA_SOLVERS", "noop")
    captcha_api_url: str = os.getenv("CAPTCHA_API_URL", "")
    captcha_api_key: str = os.getenv("CAPTCHA_API_KEY", "")

    # Email verification config.
    email_verifier: str = os.getenv("EMAIL_VERIFIER", "noop")
    mailpit_url: str = os.getenv("MAILPIT_URL", "http://mailpit:8025")
    email_domain: str = os.getenv("EMAIL_DOMAIN", "verify.example.com")


config = Config()
