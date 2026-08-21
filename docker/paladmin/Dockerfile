# PalAdmin 面板镜像

# ---- 前端构建 ----
FROM node:20-bookworm-slim AS web-builder
WORKDIR /src/web
# 国内 npm 镜像加速
RUN npm config set registry https://registry.npmmirror.com
COPY web/package*.json ./
RUN npm install --no-audit --no-fund
COPY web/ ./
RUN npm run build

# ---- Go 后端构建 ----
FROM golang:1.22-bookworm AS go-builder
# 国内 Go 模块代理
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /src/web/dist ./web/dist
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/paladmin .

# ---- 存档解析器（Python，用 PyInstaller 打包为单文件）----
# 在目标架构构建，保证 sav_cli 与镜像架构一致（amd64/arm64）
FROM python:3.12-bookworm AS savcli
WORKDIR /sav
# 国内 pip 镜像加速
RUN pip config set global.index-url https://pypi.tuna.tsinghua.edu.cn/simple
COPY module/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt pyinstaller
COPY module/ .
RUN pyinstaller --onefile --name sav_cli sav_cli.py

# ---- 运行镜像 ----
FROM debian:bookworm-slim

# 换阿里云 apt 镜像（国内构建加速）。用 http，避免 slim 镜像缺 ca-certificates 导致 https 失败。
RUN set -eux; \
    if [ -f /etc/apt/sources.list.d/debian.sources ]; then \
        sed -i -E 's|https?://deb.debian.org|http://mirrors.aliyun.com|g; s|https?://security.debian.org|http://mirrors.aliyun.com|g' /etc/apt/sources.list.d/debian.sources; \
    fi; \
    if [ -f /etc/apt/sources.list ]; then \
        sed -i -E 's|https?://deb.debian.org|http://mirrors.aliyun.com|g; s|https?://security.debian.org|http://mirrors.aliyun.com|g' /etc/apt/sources.list; \
    fi

# 安装 ca-certificates（HTTPS）和 Docker CLI（通过 docker.sock 管控游戏服容器）
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl docker.io && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=go-builder /out/paladmin /app/paladmin
# 前端静态资源（主程序无 go:embed，运行时从 /app/web/dist 提供）
COPY --from=web-builder /src/web/dist /app/web/dist
# 存档解析器
COPY --from=savcli /sav/dist/sav_cli /app/sav_cli
RUN chmod +x /app/sav_cli
COPY docker/paladmin/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# 存档解析器路径（容器内固定位置）
ENV SAVE__DECODE_PATH=/app/sav_cli

EXPOSE 8190
VOLUME ["/data"]
# 挂载宿主机 docker.sock 以管控游戏服容器
VOLUME ["/var/run/docker.sock"]

ENTRYPOINT ["/entrypoint.sh"]
