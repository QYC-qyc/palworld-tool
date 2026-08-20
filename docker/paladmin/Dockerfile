# PalAdmin 面板镜像

# ---- 前端构建 ----
FROM node:20-bookworm-slim AS web-builder
WORKDIR /src/web
COPY web/package*.json ./
RUN npm install --no-audit --no-fund
COPY web/ ./
RUN npm run build

# ---- Go 后端构建 ----
FROM golang:1.22-bookworm AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /src/web/dist ./web/dist
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/paladmin .

# ---- 运行镜像 ----
FROM debian:bookworm-slim

# 安装 ca-certificates（HTTPS）和 Docker CLI（通过 docker.sock 管控游戏服容器）
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl docker.io && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=go-builder /out/paladmin /app/paladmin
COPY docker/paladmin/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 8190
VOLUME ["/data"]
# 挂载宿主机 docker.sock 以管控游戏服容器
VOLUME ["/var/run/docker.sock"]

ENTRYPOINT ["/entrypoint.sh"]
