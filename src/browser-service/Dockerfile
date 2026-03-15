FROM python:3.12-slim

WORKDIR /app

# Install system deps for Playwright.
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install Python deps.
COPY pyproject.toml .
RUN pip install --no-cache-dir . && \
    playwright install --with-deps chromium

# Copy application.
COPY src/ src/

EXPOSE 8091

CMD ["uvicorn", "browser_service.main:app", "--host", "0.0.0.0", "--port", "8091"]
