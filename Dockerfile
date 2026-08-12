# ---- 前端构建 ----
FROM node:20-alpine AS webbuilder
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm install --no-audit --no-fund
COPY web/ ./
RUN npm run build

# ---- 后端构建 ----
FROM golang:1.22-bookworm AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=webbuilder /web/dist ./web/dist
# CGO_ENABLED=0 纯静态
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w -X main.version=docker" -o /out/paladmin .

# ---- 存档解析器（Python，用 PyInstaller 打包为单文件）----
FROM python:3.12-bookworm AS savcli
WORKDIR /sav
COPY module/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt pyinstaller
COPY module/ .
RUN pyinstaller --onefile --name sav_cli sav_cli.py

# ---- 运行时 ----
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates tini \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app

COPY --from=backend /out/paladmin /app/paladmin
COPY --from=savcli /sav/dist/sav_cli /app/sav_cli
COPY --from=webbuilder /web/dist /app/web/dist
COPY config.yaml /app/config.yaml
COPY data /app/data

# 运行时目录
RUN mkdir -p /app/backups /app/evidence /app/logs
ENV SAVE__DECODE_PATH=/app/sav_cli

EXPOSE 8190
ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["/app/paladmin", "--config", "/app/config.yaml"]
