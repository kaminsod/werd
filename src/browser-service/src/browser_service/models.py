from pydantic import BaseModel


class ActionOptions(BaseModel):
    timeout_secs: int = 30
    headless: bool = True
    proxy: str = ""
    screenshot_on_error: bool = True


class PublishRequest(BaseModel):
    platform: str
    credentials: dict
    content: str
    options: ActionOptions = ActionOptions()


class ReadRequest(BaseModel):
    platform: str
    credentials: dict
    target: str = ""
    options: ActionOptions = ActionOptions()


class ValidateRequest(BaseModel):
    platform: str
    credentials: dict
    options: ActionOptions = ActionOptions()


class PublishResponse(BaseModel):
    success: bool
    post_id: str = ""
    url: str = ""
    error: str = ""
    screenshot_b64: str | None = None


class ReadItem(BaseModel):
    title: str = ""
    url: str = ""
    content: str = ""
    author: str = ""
    date: str = ""


class ReadResponse(BaseModel):
    success: bool
    items: list[ReadItem] = []
    error: str = ""


class ValidateResponse(BaseModel):
    success: bool
    error: str = ""


class HealthResponse(BaseModel):
    status: str = "ok"
