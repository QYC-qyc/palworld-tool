#!/usr/bin/env bash
# PalAdmin Ubuntu 22.04 一键安装（systemd 裸机部署）
# 用法: sudo bash install.sh
set -euo pipefail

APP_DIR="/opt/paladmin"
APP_USER="paladmin"
SERVICE_NAME="paladmin"

echo "==> 创建运行用户 $APP_USER"
id -u "$APP_USER" >/dev/null 2>&1 || useradd --system --create-home --home-dir "$APP_DIR" --shell /usr/sbin/nologin "$APP_USER"

echo "==> 安装系统依赖"
export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y --no-install-recommends ca-certificates python3 python3-pip python3-venv

echo "==> 创建目录 $APP_DIR"
mkdir -p "$APP_DIR"
# 复制后端二进制（请先在构建机用 build.sh 产出 paladmin，或从 release 下载）
if [ -f ./paladmin ]; then cp ./paladmin "$APP_DIR/"; fi
# 复制前端与数据
[ -d ./web ] && cp -r ./web "$APP_DIR/" || true
[ -d ./data ] && cp -r ./data "$APP_DIR/" || true
[ -f ./config.yaml ] && cp ./config.yaml "$APP_DIR/" || true

echo "==> 安装 sav_cli（Python 存档解析器）"
if [ -d ./module ]; then
    mkdir -p "$APP_DIR/module"
    cp -r ./module/* "$APP_DIR/module/"
    python3 -m pip install --break-system-packages -r "$APP_DIR/module/requirements.txt" || \
        python3 -m pip install -r "$APP_DIR/module/requirements.txt"
    # 直接用 python 脚本作为 sav_cli 入口
    cat > "$APP_DIR/sav_cli" <<'EOF'
#!/usr/bin/env bash
exec python3 /opt/paladmin/module/sav_cli.py "$@"
EOF
    chmod +x "$APP_DIR/sav_cli"
fi

echo "==> 权限"
chown -R "$APP_USER:$APP_USER" "$APP_DIR"
mkdir -p "$APP_DIR/backups" "$APP_DIR/evidence" "$APP_DIR/logs"
chown -R "$APP_USER:$APP_USER" "$APP_DIR/backups" "$APP_DIR/evidence" "$APP_DIR/logs"

echo "==> 安装 systemd 服务"
cp ./deploy/${SERVICE_NAME}.service /etc/systemd/system/${SERVICE_NAME}.service
systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl restart "$SERVICE_NAME"

echo "==> 完成！查看状态: systemctl status $SERVICE_NAME"
echo "    默认监听 :8080，请务必修改 config.yaml 中的 web.password"
