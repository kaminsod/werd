FROM golang:1.23-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o monitor-reddit ./cmd/monitor-reddit

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /build/monitor-reddit /usr/local/bin/monitor-reddit

USER nonroot:nonroot

ENTRYPOINT ["monitor-reddit"]
