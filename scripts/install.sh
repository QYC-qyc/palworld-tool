#!/usr/bin/env bash
# PalAdmin 一键安装脚本（二进制 + systemd，直接安装到宿主机）
# 用法（国内服务器推荐用 Gitee 源拉脚本）:
#   curl -fsSL https://gitee.com/QYC-qyc/palworld-tool/raw/main/scripts/install.sh | sudo bash
# GitHub 源:
#   curl -fsSL https://raw.githubusercontent.com/QYC-qyc/palworld-tool/main/scripts/install.sh | sudo bash
set -e

INSTALL_DIR="/opt/paladmin"
DATA_DIR="/var/lib/paladmin"
SERVICE="paladmin"
REPO="QYC-qyc/palworld-tool"

# 面板以 root 运行，以便通过 SteamCMD 向任意用户指定目录安装/更新游戏服，
# 避免因目录属主不是面板用户而需要手动 chown 赋权。
if [ "$(id -u)" -ne 0 ]; then
  echo "请使用 root 运行本脚本（sudo bash install.sh）"
  exit 1
fi

echo "==> 创建目录"
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
  "https://ghfast.top/https://github.com"
  "https://gh-proxy.com/https://github.com"
  "https://ghproxy.net/https://github.com"
  "https://github.com"
)

echo "==> 下载 $ASSET（多镜像自动重试，显示进度）"
TMP="$(mktemp -d)"
DOWNLOADED=0
for BASE in "${MIRRORS[@]}"; do
  URL="$BASE/$REPO/releases/latest/download/$ASSET"
  echo "  -> 尝试: $BASE"
  if command -v aria2c >/dev/null 2>&1; then
    if aria2c -x16 -s16 -k1M --console-log-level=warn --summary-interval=3 \
        --connect-timeout=10 --timeout=30 -d "$TMP" -o "$ASSET" "$URL"; then
      DOWNLOADED=1; break
    fi
  else
    # -f 失败返回非零、-L 跟随重定向、--progress-bar 显示进度
    if curl -fL --progress-bar --retry 1 --connect-timeout 10 --max-time 180 \
        -o "$TMP/$ASSET" "$URL"; then
      DOWNLOADED=1; break
    fi
  fi
  echo "  失败，尝试下一个镜像..."
done

if [ "$DOWNLOADED" != "1" ]; then
  echo "==> 所有镜像均下载失败"
  echo "    1) 配置代理后重试: export https_proxy=http://你的代理IP:端口"
  echo "    2) 或在能访问 GitHub 的机器手动下载后上传到服务器："
  echo "       https://github.com/$REPO/releases/latest/download/$ASSET"
  echo "       上传后执行: tar -xzf $ASSET -C /tmp/paladmin && cp /tmp/paladmin/paladmin $INSTALL_DIR/"
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

echo "==> 安装 systemd 服务"
cp "$TMP/paladmin.service" /etc/systemd/system/ 2>/dev/null || cat > /etc/systemd/system/paladmin.service <<EOF
[Unit]
Description=PalAdmin Panel
After=network.target
[Service]
Type=simple
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
