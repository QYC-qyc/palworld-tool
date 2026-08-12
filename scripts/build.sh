#!/usr/bin/env bash
# 在 Linux 构建机（Ubuntu 22.04）上编译 paladmin 与打包发布物
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${VERSION:-$(git describe --tags --always 2>/dev/null || echo dev)}"
OUT="dist/paladmin"

echo "==> 构建前端"
cd web
npm install --no-audit --no-fund
npm run build
cd ..

echo "==> 编译后端 (CGO_ENABLED=0)"
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$OUT/paladmin" .

echo "==> 打包"
mkdir -p "$OUT"
cp config.yaml "$OUT/"
cp -r web/dist "$OUT/web/"
cp -r data "$OUT/"
cp -r module "$OUT/"
cp -r deploy "$OUT/"
cp scripts/install.sh "$OUT/install.sh"
chmod +x "$OUT/install.sh"

tar -czf "dist/paladmin-${VERSION}-linux-amd64.tar.gz" -C dist paladmin
echo "==> 完成: dist/paladmin-${VERSION}-linux-amd64.tar.gz"
