FROM golang:1.23-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o monitor-hn ./cmd/monitor-hn

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /build/monitor-hn /usr/local/bin/monitor-hn

USER nonroot:nonroot

ENTRYPOINT ["monitor-hn"]
