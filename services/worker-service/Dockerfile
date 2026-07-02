# --- build stage ---
FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY services/trip-service services/trip-service
COPY services/worker-service services/worker-service

WORKDIR /src/services/worker-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker-service ./cmd/worker

# --- runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget && adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/worker-service /usr/local/bin/worker-service
COPY --from=builder /src/services/trip-service/migrations /app/migrations

USER appuser
EXPOSE 8090

ENTRYPOINT ["worker-service"]
