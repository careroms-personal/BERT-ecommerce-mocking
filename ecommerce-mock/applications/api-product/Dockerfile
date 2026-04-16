FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install dependencies first (layer cache)
COPY go.mod ./
RUN go mod download || true

COPY . .
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /api-product .

# ─────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app
COPY --from=builder /api-product .

EXPOSE 8081

CMD ["./api-product"]
