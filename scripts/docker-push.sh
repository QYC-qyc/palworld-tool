#!/usr/bin/env bash
# 构建并推送 PalAdmin 镜像到 Gitee 容器镜像仓库
# 用法：
#   export DOCKER_USERNAME=qyc-qyc
#   export DOCKER_PASSWORD=你的Gitee令牌
#   bash scripts/docker-push.sh [tag]
set -euo pipefail
cd "$(dirname "$0")/.."

REGISTRY="gitee.com"
NAMESPACE="${DOCKER_USERNAME:-qyc-qyc}"
IMAGE="paladmin"
TAG="${1:-latest}"
FULL="${REGISTRY}/${NAMESPACE}/${IMAGE}:${TAG}"

echo "==> 构建镜像 ${FULL} (linux/amd64)"
docker build --platform linux/amd64 -t "${FULL}" .

echo "==> 登录 ${REGISTRY}"
if [ -n "${DOCKER_PASSWORD:-}" ]; then
  echo "$DOCKER_PASSWORD" | docker login "$REGISTRY" -u "$NAMESPACE" --password-stdin
else
  echo "请先 docker login $REGISTRY"
fi

echo "==> 推送 ${FULL}"
docker push "${FULL}"
echo "==> 完成"
