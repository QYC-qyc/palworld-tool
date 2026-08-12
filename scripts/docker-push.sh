#!/usr/bin/env bash
# 构建并推送 PalAdmin 镜像到 Gitee 容器镜像仓库
# 镜像仓库：https://gitee.com/QYC-qyc/docker-palworld-tool
# 镜像地址：gitee.com/qyc-qyc/docker-palworld-tool:<tag>
#
# 用法：
#   export DOCKER_USERNAME=QYC-qyc
#   export DOCKER_PASSWORD=你的Gitee私人令牌（容器镜像仓库写权限）
#   bash scripts/docker-push.sh [tag]
set -euo pipefail
cd "$(dirname "$0")/.."

REGISTRY="gitee.com"
NAMESPACE="${DOCKER_USERNAME:-QYC-qyc}"
IMAGE="docker-palworld-tool"
TAG="${1:-latest}"
# Gitee 镜像路径命名空间全小写
FULL="${REGISTRY}/${NAMESPACE,,}/${IMAGE}:${TAG}"

echo "==> 构建镜像 ${FULL} (linux/amd64)"
docker build --platform linux/amd64 -t "${FULL}" .

echo "==> 登录 ${REGISTRY}"
if [ -n "${DOCKER_PASSWORD:-}" ]; then
  echo "$DOCKER_PASSWORD" | docker login "$REGISTRY" -u "$NAMESPACE" --password-stdin
else
  docker login "$REGISTRY"
fi

echo "==> 推送 ${FULL}"
docker push "${FULL}"
echo "==> 完成"
echo "服务器拉取: docker pull ${FULL}"
