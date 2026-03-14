.PHONY: help dev build test lint clean
.PHONY: build-api build-web build-monitors
.PHONY: test-api test-web
.PHONY: dev-api dev-web
.PHONY: compose-up compose-down compose-ps compose-logs compose-check-dns compose-health
.PHONY: generate-secrets

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

# ── Scripts ──

generate-secrets: ## Generate secrets and write to .env
	./tools/generate-secrets.sh

# ── Clean ──

clean: ## Remove build artifacts
	$(MAKE) -C src/go/api clean
	$(MAKE) -C src/web clean
	$(MAKE) -C src/go/monitor-reddit clean
	$(MAKE) -C src/go/monitor-hn clean
