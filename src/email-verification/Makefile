.PHONY: run stop logs test deploy setup-server

COMPOSE_CMD := podman-compose

run: ## Start Mailpit locally
	$(COMPOSE_CMD) up -d

stop: ## Stop Mailpit
	$(COMPOSE_CMD) down

logs: ## Tail Mailpit logs
	$(COMPOSE_CMD) logs -f

test: ## Send test email and verify via API
	./scripts/test-email.sh localhost

deploy: ## Push config and restart on production server
	./scripts/deploy.sh

setup-server: ## First-time VPS provisioning (podman, UFW, Mailpit)
	./scripts/setup-server.sh
