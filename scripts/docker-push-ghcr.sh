#!/usr/bin/env bash
# 构建并推送 PalAdmin 镜像到 GitHub Container Registry (GHCR)
# 用法：
#   export GITHUB_USER=qyc-qyc            # GitHub 用户名（小写）
#   export GITHUB_TOKEN=你的GitHubToken    # 需要 write:packages 权限
#   bash scripts/docker-push-ghcr.sh [tag]
set -euo pipefail
cd "$(dirname "$0")/.."

REGISTRY="ghcr.io"
NAMESPACE="${GITHUB_USER:-qyc-qyc}"
IMAGE="paladmin"
TAG="${1:-latest}"
# GHCR 镜像名要求全小写
FULL="${REGISTRY}/${NAMESPACE,,}/${IMAGE}:${TAG}"

echo "==> 构建镜像 ${FULL} (linux/amd64)"
docker build --platform linux/amd64 -t "${FULL}" .

echo "==> 登录 ${REGISTRY}"
if [ -n "${GITHUB_TOKEN:-}" ]; then
  echo "$GITHUB_TOKEN" | docker login "$REGISTRY" -u "$NAMESPACE" --password-stdin
else
  docker login "$REGISTRY"
fi

echo "==> 推送 ${FULL}"
docker push "${FULL}"
echo "==> 完成"
echo "服务器拉取: docker pull ${FULL}"
