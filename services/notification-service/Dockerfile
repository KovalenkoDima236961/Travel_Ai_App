# --- build stage ---
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/notification-service ./cmd/server

# --- runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates && adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/notification-service /usr/local/bin/notification-service
COPY --from=builder /app/migrations /app/migrations

USER appuser
EXPOSE 8086

ENTRYPOINT ["notification-service"]
