#!/bin/bash
# 国内加速拉取 PalAdmin Docker 镜像（GHCR 镜像）
#
# 用法:
#   ./scripts/pull-cn.sh              # 拉取 latest
#   ./scripts/pull-cn.sh v3.0.1       # 拉取指定版本
#   MIRROR=ghcr.dockerproxy.net ./scripts/pull-cn.sh  # 使用备用加速源
#
# 拉取后会自动 retag 为 ghcr.io/...，docker compose up 可直接使用本地镜像。

set -e

VERSION="${1:-latest}"
MIRROR="${MIRROR:-ghcr.nju.edu.cn}"
OWNER="qyc-qyc"
IMAGES=("paladmin" "palworld-gameserver")

# 备选加速源（主源失败时自动尝试）
FALLBACKS=("ghcr.dockerproxy.net" "ghcr.nju.edu.cn")

echo "================================================"
echo " PalAdmin 镜像拉取（国内加速）"
echo " 版本:  $VERSION"
echo " 加速源: $MIRROR"
echo "================================================"
echo ""

pull_image() {
    local image="$1"
    local src="${MIRROR}/${OWNER}/${image}:${VERSION}"
    local dst="ghcr.io/${OWNER}/${image}:${VERSION}"
    local dst_latest="ghcr.io/${OWNER}/${image}:latest"

    echo ">>> 拉取 ${src}"
    if ! docker pull "$src"; then
        echo "!!! 主源 $MIRROR 失败，尝试备选源..."
        for fb in "${FALLBACKS[@]}"; do
            [ "$fb" = "$MIRROR" ] && continue
            echo ">>> 尝试 $fb"
            if docker pull "${fb}/${OWNER}/${image}:${VERSION}"; then
                src="${fb}/${OWNER}/${image}:${VERSION}"
                break
            fi
        done
        if ! docker image inspect "$src" >/dev/null 2>&1; then
            echo "!!! 所有加速源均失败，请检查网络或稍后重试"
            return 1
        fi
    fi

    echo ">>> 重标记为 $dst"
    docker tag "$src" "$dst"
    if [ "$VERSION" != "latest" ]; then
        docker tag "$src" "$dst_latest"
        echo ">>> 同时标记为 $dst_latest"
    fi
    echo ""
}

for img in "${IMAGES[@]}"; do
    pull_image "$img"
done

echo "================================================"
echo " 拉取完成！"
echo ""
echo " 启动命令:"
echo "   docker compose up -d"
echo ""
echo " 查看状态:"
echo "   docker compose ps"
echo "================================================"
