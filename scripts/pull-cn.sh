#!/bin/bash
# 拉取 PalWorld Panel Docker 镜像
#
# 默认从阿里云 ACR 拉取（国内高速，仓库已公开无需登录）。
#
# 用法:
#   ./scripts/pull-cn.sh              # 拉取 latest
#   ./scripts/pull-cn.sh v3.0.1       # 拉取指定版本
#   USE_GHCR=1 ./scripts/pull-cn.sh   # 改从 GHCR 拉取（海外服务器）
#
# 拉取后自动 retag 为 docker-compose.yml 使用的镜像名，可直接 docker compose up -d。

set -e

VERSION="${1:-latest}"
IMAGES=("palworld-panel" "palworld-gameserver")

# 阿里云 ACR（默认，国内高速）
ALIYUN_REGISTRY="crpi-pwq7gsi7qm6vv08p.cn-chengdu.personal.cr.aliyuncs.com/qyc_pal"
# GHCR（海外备用）
GHCR_REGISTRY="ghcr.io/qyc-qyc"

if [ -n "$USE_GHCR" ]; then
    REGISTRY="$GHCR_REGISTRY"
else
    REGISTRY="$ALIYUN_REGISTRY"
fi

echo "================================================"
echo " PalWorld Panel 镜像拉取"
echo " 版本:  $VERSION"
echo " 镜像源: $REGISTRY"
echo "================================================"
echo ""

for image in "${IMAGES[@]}"; do
    src="${REGISTRY}/${image}:${VERSION}"
    echo ">>> 拉取 ${src}"
    docker pull "$src"

    # 标记为 compose 使用的名字
    docker tag "$src" "${ALIYUN_REGISTRY}/${image}:${VERSION}"
    if [ "$VERSION" != "latest" ]; then
        docker tag "$src" "${ALIYUN_REGISTRY}/${image}:latest"
    fi
    echo ""
done

echo "================================================"
echo " 拉取完成！启动命令:"
echo "   docker compose up -d"
echo ""
echo " 查看状态:"
echo "   docker compose ps"
echo "================================================"
