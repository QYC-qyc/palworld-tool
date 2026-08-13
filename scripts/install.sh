#!/usr/bin/env bash
# PalAdmin 一键安装脚本（二进制 + systemd，直接安装到宿主机）
# 用法:
#   curl -fsSL https://gitee.com/QYC-qyc/palworld-tool/raw/main/scripts/install.sh | sudo bash
set -e

INSTALL_DIR="/opt/paladmin"
DATA_DIR="/var/lib/paladmin"
SERVICE="paladmin"
REPO="QYC-qyc/palworld-tool"
# 优先从 Gitee 下载，失败则用 GitHub
BASE_GITEE="https://gitee.com/${REPO}/releases/latest/download"
BASE_GH="https://github.com/${REPO}/releases/latest/download"

echo "==> 创建用户与目录"
id -u paladmin &>/dev/null || useradd -r -s /usr/sbin/nologin paladmin
mkdir -p "$INSTALL_DIR" "$DATA_DIR/backups" "$DATA_DIR/evidence" "$DATA_DIR/logs"

echo "==> 检测架构"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ASSET="paladmin_linux_amd64.tar.gz" ;;
  aarch64|arm64) ASSET="paladmin_linux_arm64.tar.gz" ;;
  *) echo "不支持的架构: $ARCH"; exit 1 ;;
esac

echo "==> 下载 $ASSET"
TMP="$(mktemp -d)"
if ! curl -fsSL -o "$TMP/$ASSET" "$BASE_GITEE/$ASSET"; then
  echo "Gitee 下载失败，尝试 GitHub..."
  curl -fsSL -o "$TMP/$ASSET" "$BASE_GH/$ASSET"
fi

echo "==> 解压安装"
tar -xzf "$TMP/$ASSET" -C "$TMP"
cp "$TMP/paladmin" "$INSTALL_DIR/"
# sav_cli 与 web 资源（若包含）
[ -f "$TMP/sav_cli" ] && cp "$TMP/sav_cli" "$INSTALL_DIR/"
[ -d "$TMP/web" ] && cp -r "$TMP/web" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/paladmin"
[ -f "$INSTALL_DIR/sav_cli" ] && chmod +x "$INSTALL_DIR/sav_cli"

# 生成默认配置
if [ ! -f "$INSTALL_DIR/config.yaml" ]; then
  cat > "$INSTALL_DIR/config.yaml" <<EOF
web:
  port: 8190
storage:
  path: "${DATA_DIR}/pst.db"
save:
  decode_path: "${INSTALL_DIR}/sav_cli"
anticheat:
  enabled: true
  mode: "external"
EOF
fi

chown -R paladmin:paladmin "$INSTALL_DIR" "$DATA_DIR"

echo "==> 安装 systemd 服务"
cp "$TMP/paladmin.service" /etc/systemd/system/ 2>/dev/null || cat > /etc/systemd/system/paladmin.service <<EOF
[Unit]
Description=PalAdmin Panel
After=network.target
[Service]
Type=simple
User=paladmin
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/paladmin --config ${INSTALL_DIR}/config.yaml
Restart=always
RestartSec=5
[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "$SERVICE"
systemctl restart "$SERVICE"

rm -rf "$TMP"
echo ""
echo "==> 安装完成！"
echo "    面板地址: http://$(hostname -I | awk '{print $1}'):8190"
echo "    配置文件: ${INSTALL_DIR}/config.yaml"
echo "    查看日志: journalctl -u paladmin -f"
