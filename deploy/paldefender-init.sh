#!/bin/bash
# PalDefender 首次启动初始化：自动创建 REST API Token
# 在 Wine 容器启动游戏前执行，确保 Tokens 目录与 token 文件存在。
set -e

PD_DIR="/palworld/Pal/Binaries/Win64"
TOKENS_DIR="${PD_DIR}/config/RESTAPI/Tokens"
TOKEN_FILE="${TOKENS_DIR}/paladmin.json"

mkdir -p "$TOKENS_DIR"

if [ ! -f "$TOKEN_FILE" ]; then
  # 优先使用环境变量 PALDEFENDER_TOKEN，否则自动生成随机串
  if [ -z "$PALDEFENDER_TOKEN" ]; then
    PALDEFENDER_TOKEN=$(head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n')
    echo "[paldefender-init] 已自动生成 REST API Token: $PALDEFENDER_TOKEN"
  else
    echo "[paldefender-init] 使用环境变量 PALDEFENDER_TOKEN"
  fi

  cat > "$TOKEN_FILE" <<EOF
{
  "Name": "PalAdmin",
  "Token": "${PALDEFENDER_TOKEN}",
  "Permissions": ["REST.*"]
}
EOF
  echo "[paldefender-init] Token 文件已创建: $TOKEN_FILE"
else
  echo "[paldefender-init] Token 文件已存在，跳过"
fi

# 同时确保 RESTConfig.json 启用 API
REST_CONFIG="${PD_DIR}/config/RESTAPI/RESTConfig.json"
if [ ! -f "$REST_CONFIG" ]; then
  mkdir -p "$(dirname "$REST_CONFIG")"
  cat > "$REST_CONFIG" <<'EOF'
{
  "Enabled": true,
  "Port": 17993
}
EOF
  echo "[paldefender-init] RESTConfig.json 已创建并启用 (端口 17993)"
fi

echo "[paldefender-init] 初始化完成"
