# --- build stage ---
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

# Build the static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/user-service ./cmd/server

# --- runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget && adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/user-service /usr/local/bin/user-service
# Migrations are applied by the service on startup.
COPY --from=builder /app/migrations /app/migrations

USER appuser
EXPOSE 8083

ENTRYPOINT ["user-service"]

