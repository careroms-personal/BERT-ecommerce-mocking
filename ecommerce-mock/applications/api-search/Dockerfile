FROM rust:1.86-alpine AS builder

# musl-dev is all we need — no OpenSSL since we use rustls
RUN apk add --no-cache musl-dev

WORKDIR /app

COPY Cargo.toml ./
COPY src ./src

RUN cargo build --release

# ─────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app
COPY --from=builder /app/target/release/api-search .

EXPOSE 8086


CMD ["./api-search"]
