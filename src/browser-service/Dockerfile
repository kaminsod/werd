FROM python:3.12-slim

WORKDIR /app

# Install system deps for Playwright/Patchright.
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy everything, then install (needs source for package discovery).
COPY . .
RUN pip install --no-cache-dir ".[captcha]" && \
    patchright install --with-deps chromium || \
    playwright install --with-deps chromium

EXPOSE 8091

CMD ["uvicorn", "browser_service.main:app", "--host", "0.0.0.0", "--port", "8091"]
