#!/usr/bin/env bash
# PalAdmin 一键安装脚本（二进制 + systemd，直接安装到宿主机）
# 用法:
#   curl -fsSL https://raw.githubusercontent.com/QYC-qyc/palworld-tool/main/scripts/install.sh | sudo bash
set -e

INSTALL_DIR="/opt/paladmin"
DATA_DIR="/var/lib/paladmin"
SERVICE="paladmin"
REPO="QYC-qyc/palworld-tool"

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

# 公共 GitHub 加速镜像（按顺序尝试，最后直连兜底）
MIRRORS=(
  "https://ghproxy.net/https://github.com"
  "https://gh-proxy.com/https://github.com"
  "https://ghfast.top/https://github.com"
  "https://mirror.ghproxy.com/https://github.com"
  "https://github.com"
)

echo "==> 下载 $ASSET（多镜像自动重试）"
TMP="$(mktemp -d)"
DOWNLOADED=0
for BASE in "${MIRRORS[@]}"; do
  URL="$BASE/$REPO/releases/latest/download/$ASSET"
  echo "  尝试: $BASE"
  if command -v aria2c >/dev/null 2>&1; then
    echo "  使用 aria2 多线程下载..."
    if aria2c -x16 -s16 -k1M --summary-interval=0 -d "$TMP" -o "$ASSET" "$URL"; then
      DOWNLOADED=1; break
    fi
  else
    if curl -fL --retry 2 --connect-timeout 10 --max-time 300 -o "$TMP/$ASSET" "$URL"; then
      DOWNLOADED=1; break
    fi
  fi
  echo "  失败，尝试下一个镜像..."
done

if [ "$DOWNLOADED" != "1" ]; then
  echo "==> 所有镜像均下载失败"
  echo "    可手动下载后放到 $TMP/$ASSET 重新运行，或配置代理后重试："
  echo "    export https_proxy=http://你的代理IP:端口"
  exit 1
fi
echo "  下载完成"

echo "==> 解压安装"
tar -xzf "$TMP/$ASSET" -C "$TMP"
cp "$TMP/paladmin" "$INSTALL_DIR/"
# sav_cli、web 前端、data 游戏数据（若包含）
[ -f "$TMP/sav_cli" ] && cp "$TMP/sav_cli" "$INSTALL_DIR/"
[ -d "$TMP/web" ] && cp -r "$TMP/web" "$INSTALL_DIR/"
[ -d "$TMP/data" ] && cp -r "$TMP/data" "$INSTALL_DIR/"
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

# 获取公网 IP（优先），失败则用本机内网 IP
PANEL_PORT=8190
SERVER_IP=$(curl -s --max-time 5 https://api.ipify.org 2>/dev/null || true)
[ -z "$SERVER_IP" ] && SERVER_IP=$(curl -s --max-time 5 https://ifconfig.me 2>/dev/null || true)
[ -z "$SERVER_IP" ] && SERVER_IP=$(hostname -I 2>/dev/null | awk '{print $1}')

echo ""
echo "==> 安装完成！"
echo "    面板地址: http://${SERVER_IP}:${PANEL_PORT}"
echo "    配置文件: ${INSTALL_DIR}/config.yaml"
echo "    查看日志: journalctl -u paladmin -f"
