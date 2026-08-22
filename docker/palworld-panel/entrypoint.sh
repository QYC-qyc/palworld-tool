#!/bin/bash
set -e

# 持久化数据目录（面板数据库、备份、日志）
mkdir -p /data/backups /data/logs
export PALADIN_DATA_DIR=/data
export PALADIN_CONTAINER=1

# Web 端口（镜像内无 config.yaml，需显式指定以匹配 compose 映射的 8190）
export WEB__PORT="${WEB__PORT:-8190}"

# 配置游戏服容器名（面板通过 Docker API 管控它）
export GAMESERVER_CONTAINER="${GAMESERVER_CONTAINER:-palworld-gameserver}"

# ---- 以下映射为 viper 配置项（分隔符 . 用 __ 表示，见 internal/config/config.go）----

# 数据库写到持久卷，避免容器重建丢失
export STORAGE__PATH="${STORAGE__PATH:-/data/pst.db}"
# 日志写到持久卷
export LOG__FILE="${LOG__FILE:-/data/logs/pst.log}"

# 游戏服官方 REST API 地址（compose 通过 GAMESERVER_URL 传入）
if [ -n "${GAMESERVER_URL}" ]; then
    export REST__ADDRESS="${GAMESERVER_URL}"
fi
# REST 密码（gameserver 启动参数中的 -RESTPassword）
if [ -n "${REST_PASSWORD}" ]; then
    export REST__PASSWORD="${REST_PASSWORD}"
fi

# PalDefender REST 跑在游戏服容器内，面板需连容器名而非 127.0.0.1
export PALDEFENDER__HOST="${PALDEFENDER__HOST:-palworld-gameserver}"

# 回档等功能的进程控制：复用同一个游戏服容器
export PROCESS__MODE=docker
export PROCESS__CONTAINER="${GAMESERVER_CONTAINER}"

exec /app/palworld-panel "$@"
