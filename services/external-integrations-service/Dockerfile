# --- build stage ---
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/external-integrations-service ./cmd/server

# --- runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget && adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/external-integrations-service /usr/local/bin/external-integrations-service

USER appuser
EXPOSE 8084

ENTRYPOINT ["external-integrations-service"]
