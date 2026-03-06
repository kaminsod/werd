FROM golang:1.23-alpine AS builder

WORKDIR /build
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o werd-api ./cmd/werd-api

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /build/werd-api /usr/local/bin/werd-api

USER nonroot:nonroot
EXPOSE 8090

ENTRYPOINT ["werd-api"]
