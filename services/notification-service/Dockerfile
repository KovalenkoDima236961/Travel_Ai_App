# --- build stage ---
FROM golang:1.26-alpine AS builder

WORKDIR /app

ARG APP_VERSION=dev
ARG GIT_SHA=unknown
ARG BUILD_TIME=unknown

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/version.Version=${APP_VERSION} -X github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/version.GitSHA=${GIT_SHA} -X github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/version.BuildTime=${BUILD_TIME}" -o /out/notification-service ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/version.Version=${APP_VERSION} -X github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/version.GitSHA=${GIT_SHA} -X github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/version.BuildTime=${BUILD_TIME}" -o /out/notification-service-migrate ./cmd/migrate

# --- runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget && adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/notification-service /usr/local/bin/notification-service
COPY --from=builder /out/notification-service-migrate /usr/local/bin/notification-service-migrate
COPY --from=builder /app/migrations /app/migrations

USER appuser
EXPOSE 8086

ENTRYPOINT ["notification-service"]
