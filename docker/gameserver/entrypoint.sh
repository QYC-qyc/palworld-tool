#!/bin/bash
# PalServer 游戏服入口脚本
# 用法: entrypoint.sh [start|stop|restart|install|update|status]

set -e

PALSERVER_DIR="/home/steam/palserver"
EXE="${PALSERVER_DIR}/Pal/Binaries/Win64/PalServer-Win64-Shipping-Cmd.exe"
STEAMCMD="/usr/games/steamcmd"

# 下载/更新游戏服
install_server() {
    echo ">>> 安装/更新 PalServer 到 ${PALSERVER_DIR}"
    mkdir -p "${PALSERVER_DIR}"
    # 首次需初始化
    ${STEAMCMD} +login anonymous +quit || true
    ${STEAMCMD} \
        +@sSteamCmdForcePlatformType windows \
        +force_install_dir "${PALSERVER_DIR}" \
        +login anonymous \
        +app_update ${STEAMAPPID} validate \
        +quit
    echo ">>> 安装完成"
}

# 启动游戏服（通过 Proton）
start_server() {
    if [ ! -f "${EXE}" ]; then
        echo ">>> 游戏服未安装，先执行安装..."
        install_server
    fi

    echo ">>> 启动 PalServer (Proton)"
    cd "${PALSERVER_DIR}"

    # 注入 PalDefender DLL（如果存在）
    WIN64_DIR="${PALSERVER_DIR}/Pal/Binaries/Win64"
    if [ -f "${WIN64_DIR}/d3d9.dll" ] && [ -f "${WIN64_DIR}/PalDefender.dll" ]; then
        echo ">>> 检测到 PalDefender，将通过 WINEDLLOVERRIDES 注入"
    else
        echo ">>> 警告：未检测到 PalDefender DLL，游戏服将以无反作弊模式启动"
        WINEDLLOVERRIDES=""
    fi

    # 通过 Proton 启动 Windows exe
    exec "${PROTONPATH}" run "${EXE}" \
        -port=8211 \
        -publiclobby \
        -useperfthreads \
        -NoAsyncLoadingThread \
        -UseMultithreadForDS
}

# 停止游戏服（发 SIGTERM 给 Proton/wineserver）
stop_server() {
    echo ">>> 停止 PalServer..."
    wineserver -k 2>/dev/null || true
    pkill -f "PalServer-Win64" 2>/dev/null || true
}

case "${1:-start}" in
    start)
        # 捕获信号，优雅停止
        trap stop_server SIGTERM SIGINT
        start_server &
        wait $!
        ;;
    install|update)
        install_server
        ;;
    stop)
        stop_server
        ;;
    restart)
        stop_server
        sleep 2
        start_server
        ;;
    status)
        if pgrep -f "PalServer-Win64" > /dev/null; then
            echo "running"
            exit 0
        else
            echo "stopped"
            exit 1
        fi
        ;;
    *)
        exec "$@"
        ;;
esac
