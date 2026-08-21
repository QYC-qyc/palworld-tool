#!/bin/bash
# PalServer 游戏服入口脚本
# 用法: entrypoint.sh [start|stop|restart|install|update|status]
#
# 路径均可通过环境变量覆盖（在 docker-compose.yml 中配置）：
#   STEAMCMD_DIR   SteamCMD 安装目录（默认 /opt/steamcmd，建议挂卷持久化）
#   PALSERVER_DIR  游戏安装目录（默认 /home/steam/palserver，已挂卷持久化）

set -e

STEAMCMD_DIR="${STEAMCMD_DIR:-/opt/steamcmd}"
PALSERVER_DIR="${PALSERVER_DIR:-/home/steam/palserver}"
EXE="${PALSERVER_DIR}/Pal/Binaries/Win64/PalServer-Win64-Shipping-Cmd.exe"
STEAMCMD="${STEAMCMD_DIR}/steamcmd.sh"
REST_PORT="${REST_PORT:-8212}"
REST_PASSWORD="${REST_PASSWORD:-}"

install_steamcmd() {
    if [ -x "${STEAMCMD}" ]; then
        return 0
    fi
    echo ">>> 首次运行：安装 SteamCMD 到 ${STEAMCMD_DIR} ..."
    mkdir -p "${STEAMCMD_DIR}"
    cd "${STEAMCMD_DIR}"
    # cloudflare 的 steamstatic CDN 在国内可访问；akamaihd 作为兜底
    curl -fSL --retry 5 --retry-delay 3 --connect-timeout 30 \
        "https://cdn.cloudflare.steamstatic.com/client/installer/steamcmd_linux.tar.gz" \
        -o steamcmd.tar.gz \
    || curl -fSL --retry 5 --retry-delay 3 --connect-timeout 30 \
        "https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz" \
        -o steamcmd.tar.gz
    test -s steamcmd.tar.gz
    tar -xzf steamcmd.tar.gz
    rm -f steamcmd.tar.gz
    chmod +x steamcmd.sh
    chmod -R a+rX "${STEAMCMD_DIR}"
    echo ">>> SteamCMD 安装完成"
}

install_server() {
    echo ">>> 安装/更新 PalServer 到 ${PALSERVER_DIR}"
    mkdir -p "${PALSERVER_DIR}"
    # 首次自更新（可能因网络失败，不阻断，app_update 时会再试）
    "${STEAMCMD}" +login anonymous +quit || true
    "${STEAMCMD}" \
        +@sSteamCmdForcePlatformType windows \
        +force_install_dir "${PALSERVER_DIR}" \
        +login anonymous \
        +app_update "${STEAMAPPID:-2394010}" validate \
        +quit
    echo ">>> 安装/更新完成"
}

start_server() {
    if [ ! -f "${EXE}" ]; then
        echo ">>> 游戏服未安装，先执行安装..."
        install_server
    fi

    echo ">>> 启动 PalServer (Proton)"
    cd "${PALSERVER_DIR}"

    # 注入 PalDefender DLL（若已安装：d3d9.dll + PalDefender.dll）
    WIN64_DIR="${PALSERVER_DIR}/Pal/Binaries/Win64"
    if [ -f "${WIN64_DIR}/d3d9.dll" ] && [ -f "${WIN64_DIR}/PalDefender.dll" ]; then
        echo ">>> 检测到 PalDefender，通过 WINEDLLOVERRIDES 注入"
        export WINEDLLOVERRIDES="d3d9=n,b"
    else
        echo ">>> 未检测到 PalDefender，以无反作弊模式启动"
        unset WINEDLLOVERRIDES
    fi

    REST_ARGS=(-RESTAPI -RESTPort="${REST_PORT}")
    if [ -n "${REST_PASSWORD}" ]; then
        REST_ARGS+=(-RESTPassword="${REST_PASSWORD}")
    else
        echo ">>> 警告：未设置 REST_PASSWORD，REST API 可能拒绝连接"
    fi

    exec "${PROTONPATH}" run "${EXE}" \
        -port=8211 \
        -publiclobby \
        -useperfthreads \
        -NoAsyncLoadingThread \
        -UseMultithreadForDS \
        "${REST_ARGS[@]}"
}

stop_server() {
    echo ">>> 停止 PalServer..."
    wineserver -k 2>/dev/null || pkill -f PalServer-Win64 || true
}

# ---- 主流程 ----

# 容器以 root 启动：先装好 SteamCMD 并确保卷权限，再降权给 steam 运行游戏
if [ "$(id -u)" = "0" ]; then
    # root 阶段：确保挂载卷可写
    mkdir -p "${STEAMCMD_DIR}" "${PALSERVER_DIR}"
    chown -R steam:steam "${STEAMCMD_DIR}" "${PALSERVER_DIR}" /home/steam 2>/dev/null || true
    install_steamcmd
    chown -R steam:steam "${STEAMCMD_DIR}" 2>/dev/null || true

    # 以 steam 身份重新执行本脚本。用环境变量传参，避免 su 下 $@ 引号问题。
    export _ENTRY_ARG="${1:-start}"
    exec su steam -c '
        STEAMCMD_DIR="'"$STEAMCMD_DIR"'" PALSERVER_DIR="'"$PALSERVER_DIR"'" \
        REST_PORT="'"$REST_PORT"'" REST_PASSWORD="'"$REST_PASSWORD"'" \
        PROTONPATH="'"$PROTONPATH"'" WINEDLLOVERRIDES="'"$WINEDLLOVERRIDES"'" \
        STEAMAPPID="'"${STEAMAPPID:-2394010}"'" XDG_RUNTIME_DIR=/tmp/runtime-steam \
        _ENTRY_ARG="'"${_ENTRY_ARG}"'" bash "'"$0"'"
    '
fi

# ---- 以下以 steam 用户运行 ----
export XDG_RUNTIME_DIR=/tmp/runtime-steam
mkdir -p "${XDG_RUNTIME_DIR}" 2>/dev/null || true

case "${_ENTRY_ARG:-start}" in
    start)
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
        if pgrep -f PalServer-Win64 >/dev/null; then
            echo "running"; exit 0
        else
            echo "stopped"; exit 1
        fi
        ;;
    *)
        exec "$@"
        ;;
esac
