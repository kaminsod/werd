.PHONY: build dev test lint clean

BIN := monitor-hn
CMD := ./cmd/monitor-hn

build:
	go build -o $(BIN) $(CMD)

dev:
	go run $(CMD)

test:
	go test ./... -v

lint:
	go vet ./...

clean:
	rm -f $(BIN)
