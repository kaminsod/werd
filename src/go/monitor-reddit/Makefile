.PHONY: build dev test lint clean

BIN := monitor-reddit
CMD := ./cmd/monitor-reddit

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
