# --- build stage ---
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/auth-service ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/auth-service-migrate ./cmd/migrate

# --- runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget && adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/auth-service /usr/local/bin/auth-service
COPY --from=builder /out/auth-service-migrate /usr/local/bin/auth-service-migrate
COPY --from=builder /app/migrations /app/migrations

USER appuser
EXPOSE 8082

ENTRYPOINT ["auth-service"]
