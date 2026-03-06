.PHONY: build dev test lint clean

BIN := monitor-reddit
CMD := ./cmd/monitor-reddit

build: ## Build binary
	go build -o $(BIN) $(CMD)

dev: ## Run with go run
	go run $(CMD)

test: ## Run tests
	go test ./... -v

lint: ## Run go vet
	go vet ./...

clean: ## Remove build artifacts
	rm -f $(BIN)
