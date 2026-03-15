import os


class Config:
    internal_api_key: str = os.getenv("INTERNAL_API_KEY", "")
    default_headless: bool = os.getenv("BROWSER_HEADLESS", "true").lower() == "true"
    default_timeout_secs: int = int(os.getenv("BROWSER_TIMEOUT_SECS", "30"))
    default_proxy: str = os.getenv("BROWSER_PROXY", "")


config = Config()
