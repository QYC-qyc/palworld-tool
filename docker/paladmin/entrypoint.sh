#!/bin/bash
set -e

# 数据目录（面板数据库、配置等）
mkdir -p /data
export PALADIN_DATA_DIR=/data

# 配置游戏服容器名（面板通过 Docker API 管控它）
export GAMESERVER_CONTAINER="${GAMESERVER_CONTAINER:-palworld-gameserver}"

exec /app/paladmin "$@"
