# --- build stage ---
FROM golang:1.26-alpine AS builder

WORKDIR /src

ARG APP_VERSION=dev
ARG GIT_SHA=unknown
ARG BUILD_TIME=unknown

COPY services/trip-service services/trip-service
COPY services/worker-service services/worker-service
COPY data/travel-knowledge data/travel-knowledge

WORKDIR /src/services/worker-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/version.Version=${APP_VERSION} -X github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/version.GitSHA=${GIT_SHA} -X github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/version.BuildTime=${BUILD_TIME}" -o /out/worker-service ./cmd/worker

# --- runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget && adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/worker-service /usr/local/bin/worker-service
COPY --from=builder /src/services/trip-service/migrations /app/migrations
COPY --from=builder /src/data/travel-knowledge /app/data/travel-knowledge

USER appuser
EXPOSE 8090

ENTRYPOINT ["worker-service"]
