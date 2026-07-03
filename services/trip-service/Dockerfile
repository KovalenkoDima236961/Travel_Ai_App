# --- build stage ---
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

# Build the static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/trip-service ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/trip-service-migrate ./cmd/migrate

# --- runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates && adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/trip-service /usr/local/bin/trip-service
COPY --from=builder /out/trip-service-migrate /usr/local/bin/trip-service-migrate
# Migrations are applied by the service on startup (golang-migrate), so they
# must be present in the runtime image. POSTGRES_MIG_PATH points here.
COPY --from=builder /app/migrations /app/migrations

USER appuser
EXPOSE 8080

ENTRYPOINT ["trip-service"]
