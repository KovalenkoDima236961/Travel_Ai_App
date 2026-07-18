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
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X github.com/KovalenkoDima236961/Travel_Ai_App/internal/version.Version=${APP_VERSION} -X github.com/KovalenkoDima236961/Travel_Ai_App/internal/version.GitSHA=${GIT_SHA} -X github.com/KovalenkoDima236961/Travel_Ai_App/internal/version.BuildTime=${BUILD_TIME}" -o /out/trip-service ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X github.com/KovalenkoDima236961/Travel_Ai_App/internal/version.Version=${APP_VERSION} -X github.com/KovalenkoDima236961/Travel_Ai_App/internal/version.GitSHA=${GIT_SHA} -X github.com/KovalenkoDima236961/Travel_Ai_App/internal/version.BuildTime=${BUILD_TIME}" -o /out/trip-service-migrate ./cmd/migrate

# --- runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget && adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/trip-service /usr/local/bin/trip-service
COPY --from=builder /out/trip-service-migrate /usr/local/bin/trip-service-migrate
# Migrations are applied by the service on startup (golang-migrate), so they
# must be present in the runtime image. POSTGRES_MIG_PATH points here.
COPY --from=builder /app/migrations /app/migrations

USER appuser
EXPOSE 8080

ENTRYPOINT ["trip-service"]
