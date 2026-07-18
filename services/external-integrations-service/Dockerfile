# --- build stage ---
FROM golang:1.26-alpine AS builder

WORKDIR /app

ARG APP_VERSION=dev
ARG GIT_SHA=unknown
ARG BUILD_TIME=unknown

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/version.Version=${APP_VERSION} -X github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/version.GitSHA=${GIT_SHA} -X github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/version.BuildTime=${BUILD_TIME}" -o /out/external-integrations-service ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/version.Version=${APP_VERSION} -X github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/version.GitSHA=${GIT_SHA} -X github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/version.BuildTime=${BUILD_TIME}" -o /out/external-integrations-service-migrate ./cmd/migrate

# --- runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget && adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/external-integrations-service /usr/local/bin/external-integrations-service
COPY --from=builder /out/external-integrations-service-migrate /usr/local/bin/external-integrations-service-migrate
COPY --from=builder /app/migrations /app/migrations

USER appuser
EXPOSE 8084

ENTRYPOINT ["external-integrations-service"]
