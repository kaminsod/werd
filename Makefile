.PHONY: help dev build test lint clean
.PHONY: build-api build-web build-monitors
.PHONY: test-api test-web test-browser-hn test-browser-reddit test-browser-bluesky test-browser-all
.PHONY: dev-api dev-web
.PHONY: compose-up compose-down compose-ps compose-logs compose-check-dns compose-health
.PHONY: integration-test integration-test-keep
.PHONY: generate-secrets
.PHONY: email-run email-stop email-test email-deploy email-setup

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ── Build ──

build: build-api build-web build-monitors ## Build all packages

build-api: ## Build the Go API server
	$(MAKE) -C src/go/api build

build-web: ## Build the React dashboard
	$(MAKE) -C src/web build

build-monitors: ## Build monitoring bots
	$(MAKE) -C src/go/monitor-reddit build
	$(MAKE) -C src/go/monitor-hn build

# ── Test ──

test: test-api test-web ## Run all tests

test-api: ## Run Go API tests
	$(MAKE) -C src/go/api test

test-web: ## Run frontend tests
	$(MAKE) -C src/web test

test-browser-hn: ## Run browser tests against real HN (no creds needed)
	cd src/browser-service && python3 -m pytest tests/test_hn.py -v --timeout=120

test-browser-reddit: ## Run browser tests against real Reddit (needs WERD_TEST_REDDIT_* env vars)
	cd src/browser-service && python3 -m pytest tests/test_reddit.py -v --timeout=120

test-browser-bluesky: ## Run browser tests against real Bluesky (needs WERD_TEST_BLUESKY_* env vars)
	cd src/browser-service && python3 -m pytest tests/test_bluesky.py -v --timeout=120

test-browser-all: ## Run all browser tests against real platforms
	cd src/browser-service && python3 -m pytest tests/ -v --timeout=120

# ── Lint ──

lint: ## Lint all packages
	$(MAKE) -C src/go/api lint
	$(MAKE) -C src/go/monitor-reddit lint
	$(MAKE) -C src/go/monitor-hn lint
	$(MAKE) -C src/web lint

# ── Dev ──

dev-api: ## Run API server in dev mode
	$(MAKE) -C src/go/api dev

dev-web: ## Run dashboard in dev mode
	$(MAKE) -C src/web dev

# ── Compose ──

COMPOSE_DIR := src/deploy/compose
COMPOSE_CMD := podman-compose -f $(COMPOSE_DIR)/docker-compose.yml --env-file $(COMPOSE_DIR)/.env

compose-up: ## Start all services via podman-compose
	$(COMPOSE_CMD) up -d

compose-down: ## Stop all services
	$(COMPOSE_CMD) down

compose-ps: ## Show running services
	$(COMPOSE_CMD) ps

compose-logs: ## Tail logs from all services
	$(COMPOSE_CMD) logs -f

compose-health: ## Show health status of all services
	@$(COMPOSE_CMD) ps

compose-check-dns: ## Verify DNS resolution between services on werd-net
	@echo "Checking DNS resolution on werd-net..."
	@for svc in postgres redis werd-api werd-dashboard caddy; do \
	  $(COMPOSE_CMD) exec caddy nslookup $$svc >/dev/null 2>&1 \
	    && printf "  %-20s OK\n" "$$svc" \
	    || printf "  %-20s FAIL\n" "$$svc"; \
	done

# ── Integration Tests ──

integration-test: ## Run Phase 1 integration tests (requires container runtime)
	./tests/integration/run.sh

integration-test-keep: ## Run integration tests, leave stack running for debugging
	WERD_TEST_KEEP=1 ./tests/integration/run.sh

# ── Scripts ──

generate-secrets: ## Generate secrets and write to .env
	./tools/generate-secrets.sh

# ── Email Verification ──

email-run: ## Start email verification server locally
	$(MAKE) -C src/email-verification run

email-stop: ## Stop email verification server
	$(MAKE) -C src/email-verification stop

email-test: ## Test email delivery and API
	$(MAKE) -C src/email-verification test

email-deploy: ## Deploy email verification to production VPS
	$(MAKE) -C src/email-verification deploy

email-setup: ## First-time VPS provisioning for email verification
	$(MAKE) -C src/email-verification setup-server

# ── Clean ──

clean: ## Remove build artifacts
	$(MAKE) -C src/go/api clean
	$(MAKE) -C src/web clean
	$(MAKE) -C src/go/monitor-reddit clean
	$(MAKE) -C src/go/monitor-hn clean
