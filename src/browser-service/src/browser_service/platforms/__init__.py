from .bluesky import BlueskyPlatform
from .reddit import RedditPlatform
from .hn import HNPlatform

PLATFORMS: dict[str, type] = {
    "bluesky": BlueskyPlatform,
    "reddit": RedditPlatform,
    "hn": HNPlatform,
}


def get_platform(name: str):
    cls = PLATFORMS.get(name)
    if cls is None:
        raise ValueError(f"unsupported platform: {name}")
    return cls()
