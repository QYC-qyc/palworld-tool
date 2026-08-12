#!/usr/bin/env bash
# 构建并推送 PalAdmin 镜像到 GitHub Container Registry (GHCR)
# 镜像：ghcr.io/qyc-qyc/palworld-tool:latest（镜像已公开，服务器拉取无需登录）
#
# 注意：推送到 GHCR 仍需登录（仅首次执行一次，凭证会被 Docker 保存）：
#   echo 你的GitHubToken | docker login ghcr.io -u QYC-qyc --password-stdin
#
# 之后直接运行：
#   bash scripts/docker-push.sh [tag]
set -euo pipefail
cd "$(dirname "$0")/.."

REGISTRY="ghcr.io"
NAMESPACE="QYC-qyc"
IMAGE="palworld-tool"
TAG="${1:-latest}"
FULL="${REGISTRY}/${NAMESPACE,,}/${IMAGE}:${TAG}"

# 未登录则提示（推送必须认证）
if ! docker buildx imagetools inspect "${REGISTRY}/${NAMESPACE,,}/${IMAGE}:latest" >/dev/null 2>&1; then
  echo "==> 尚未登录 GHCR，请先执行一次："
  echo "    echo 你的GitHubToken | docker login ghcr.io -u ${NAMESPACE} --password-stdin"
  exit 1
fi

echo "==> 构建镜像 ${FULL} (linux/amd64)"
docker build --platform linux/amd64 -t "${FULL}" .

echo "==> 推送 ${FULL}"
docker push "${FULL}"
echo "==> 完成"
echo "服务器拉取: docker pull ${FULL}"
