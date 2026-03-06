.PHONY: build dev test lint clean migrate generate

BIN := werd-api
CMD := ./cmd/werd-api

build: ## Build the API server binary
	go build -o $(BIN) $(CMD)

dev: ## Run in development mode with auto-reload
	go run $(CMD)

test: ## Run tests
	go test ./... -v

lint: ## Run linter
	go vet ./...

clean: ## Remove build artifacts
	rm -f $(BIN)

migrate-up: ## Run database migrations up
	goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down: ## Roll back last migration
	goose -dir migrations postgres "$$DATABASE_URL" down

migrate-create: ## Create a new migration (usage: make migrate-create NAME=create_users)
	goose -dir migrations create $(NAME) sql

generate: ## Generate sqlc code from queries
	sqlc generate
