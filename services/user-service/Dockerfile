# --- build stage ---
FROM golang:1.26-alpine AS builder

WORKDIR /app

ARG APP_VERSION=dev
ARG GIT_SHA=unknown
ARG BUILD_TIME=unknown

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

# Build the static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/version.Version=${APP_VERSION} -X github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/version.GitSHA=${GIT_SHA} -X github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/version.BuildTime=${BUILD_TIME}" -o /out/user-service ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/version.Version=${APP_VERSION} -X github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/version.GitSHA=${GIT_SHA} -X github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/version.BuildTime=${BUILD_TIME}" -o /out/user-service-migrate ./cmd/migrate

# --- runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget && adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/user-service /usr/local/bin/user-service
COPY --from=builder /out/user-service-migrate /usr/local/bin/user-service-migrate
# Migrations are applied by the service on startup.
COPY --from=builder /app/migrations /app/migrations

USER appuser
EXPOSE 8083

ENTRYPOINT ["user-service"]
