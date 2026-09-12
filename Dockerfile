FROM golang:alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0  \
    GOARCH="amd64" \
    GOOS=linux

WORKDIR /build
COPY . .
RUN go mod tidy
RUN go build --ldflags "-extldflags -static" -o main .

# Optional SPA stage: docker build --build-arg BUILD_FRONTEND=1
ARG BUILD_FRONTEND=0
FROM node:20-alpine AS frontend
ARG BUILD_FRONTEND=0
WORKDIR /frontend
COPY html/package*.json ./
RUN if [ "$BUILD_FRONTEND" = "1" ]; then npm ci; else mkdir -p /frontend/dist && touch /frontend/dist/.keep; fi
COPY html/ ./
RUN if [ "$BUILD_FRONTEND" = "1" ]; then npm run build; else mkdir -p /frontend/dist && touch /frontend/dist/.keep; fi

FROM alpine:latest

# 安装必要的工具（用于健康检查）
RUN apk add --no-cache ca-certificates tzdata curl wget

WORKDIR /www

COPY --from=builder /build/main /www/
COPY --from=builder /build/database/ /www/database/
COPY --from=builder /build/public/ /www/public/
COPY --from=builder /build/storage/ /www/storage/
COPY --from=builder /build/resources/ /www/resources/
# When BUILD_FRONTEND=1, Vue dist is copied into public/admin for same-origin SPA serving.
COPY --from=frontend /frontend/dist/ /www/public/admin/
# Runtime configuration must be injected by docker compose, Kubernetes secrets,
# or environment variables. Do not bake .env into the image.

# 复制启动脚本
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

# 健康检查：就绪探针（DB/Redis），见 GET /ready
HEALTHCHECK --interval=10s --timeout=3s --start-period=60s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:3000/ready || exit 1

EXPOSE 3000

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD []
