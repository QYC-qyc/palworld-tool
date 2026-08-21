#!/bin/bash
# PalServer 游戏服入口脚本（方案 A：容器常驻，游戏进程由面板控制）
#
# 用法（通常由面板通过 docker exec 调用）：
#   entrypoint.sh run       容器默认命令：常驻并守护游戏进程（崩溃自动重启）
#   entrypoint.sh start     启动游戏（交给守护进程拉起）
#   entrypoint.sh stop      停止游戏（不重启）
#   entrypoint.sh restart   重启游戏
#   entrypoint.sh status    查看游戏是否在运行（running/stopped）
#   entrypoint.sh install   安装/更新游戏（SteamCMD app_update）
#   entrypoint.sh update    同 install
#
# 路径均可通过环境变量覆盖：
#   STEAMCMD_DIR   SteamCMD 目录（默认 /opt/steamcmd，挂卷持久化）
#   PALSERVER_DIR  游戏安装目录（默认 /home/steam/palserver，挂卷持久化）

set -e

STEAMCMD_DIR="${STEAMCMD_DIR:-/opt/steamcmd}"
PALSERVER_DIR="${PALSERVER_DIR:-/home/steam/palserver}"
EXE="${PALSERVER_DIR}/Pal/Binaries/Win64/PalServer-Win64-Shipping-Cmd.exe"
STEAMCMD="${STEAMCMD_DIR}/steamcmd.sh"
REST_PORT="${REST_PORT:-8212}"
REST_PASSWORD="${REST_PASSWORD:-}"

# 游戏工作目录与 PID / 控制文件
RUN_DIR="/tmp/palworld"
PID_FILE="${RUN_DIR}/palserver.pid"
STOP_FILE="${RUN_DIR}/manual.stop"
LOG_FILE="${PALSERVER_DIR}/palserver.log"

install_steamcmd() {
    if [ -x "${STEAMCMD}" ]; then
        return 0
    fi
    echo ">>> 首次运行：安装 SteamCMD 到 ${STEAMCMD_DIR} ..."
    mkdir -p "${STEAMCMD_DIR}" "${RUN_DIR}"
    cd "${STEAMCMD_DIR}"
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
    mkdir -p "${PALSERVER_DIR}" "${RUN_DIR}"
    "${STEAMCMD}" +login anonymous +quit || true
    "${STEAMCMD}" \
        +@sSteamCmdForcePlatformType windows \
        +force_install_dir "${PALSERVER_DIR}" \
        +login anonymous \
        +app_update "${STEAMAPPID:-2394010}" validate \
        +quit
    echo ">>> 安装/更新完成"
}

# 实际启动游戏进程（前台运行，由守护进程调用）
launch_game() {
    cd "${PALSERVER_DIR}"

    # 诊断：确认游戏文件存在
    if [ ! -f "${EXE}" ]; then
        echo "!!! 找不到游戏可执行文件: ${EXE}"
        return 1
    fi
    echo ">>> 游戏可执行文件: $(ls -lh "${EXE}" | awk '{print $5, $9}')"

    # 首次启动：从 DefaultPalWorldSettings.ini 生成配置
    INI_DIR="${PALSERVER_DIR}/Pal/Saved/Config/WindowsServer"
    INI_FILE="${INI_DIR}/PalWorldSettings.ini"
    mkdir -p "${INI_DIR}"
    if [ ! -f "${INI_FILE}" ]; then
        if [ -f "${PALSERVER_DIR}/DefaultPalWorldSettings.ini" ]; then
            cp "${PALSERVER_DIR}/DefaultPalWorldSettings.ini" "${INI_FILE}"
        else
            # 兜底：创建最小配置，确保游戏能启动
            cat > "${INI_FILE}" <<'INIEOF'
[/Script/Pal.PalGameWorldSettings]
OptionSettings=(ServerName="Palworld Server",ServerDescription="",AdminPassword="",ServerPassword="",MaxPlayerNum=32)
INIEOF
        fi
        chown steam:steam "${INI_FILE}" 2>/dev/null || true
    fi

    # 确保 REST API 在 ini 中启用。
    # PalWorldSettings.ini 格式：OptionSettings=(Key=Value,Key=Value,...)
    # REST 只能通过 ini 配置，不能用命令行参数（传了会导致退出码 3）。
    if [ -f "${INI_FILE}" ]; then
        # RESTAPIEnabled=True
        if grep -q "RESTAPIEnabled" "${INI_FILE}"; then
            sed -i 's/RESTAPIEnabled=[^,)]*/RESTAPIEnabled=True/' "${INI_FILE}"
        else
            sed -i "s/OptionSettings=(/OptionSettings=(RESTAPIEnabled=True,/" "${INI_FILE}"
        fi
        # RESTAPIPort
        if grep -q "RESTAPIPort" "${INI_FILE}"; then
            sed -i "s/RESTAPIPort=[^,)]*/RESTAPIPort=${REST_PORT}/" "${INI_FILE}"
        else
            sed -i "s/OptionSettings=(/OptionSettings=(RESTAPIPort=${REST_PORT},/" "${INI_FILE}"
        fi
        # AdminPassword（REST 鉴权用）
        if [ -n "${REST_PASSWORD}" ]; then
            if grep -q "AdminPassword" "${INI_FILE}"; then
                sed -i "s/AdminPassword=\"[^\"]*\"/AdminPassword=\"${REST_PASSWORD}\"/" "${INI_FILE}"
            else
                sed -i "s/OptionSettings=(/OptionSettings=(AdminPassword=\"${REST_PASSWORD}\",/" "${INI_FILE}"
            fi
        fi
        chown steam:steam "${INI_FILE}" 2>/dev/null || true
        echo ">>> 已在配置中启用 REST API（端口 ${REST_PORT}）"
    fi

    # PalDefender 注入
    WIN64_DIR="${PALSERVER_DIR}/Pal/Binaries/Win64"
    if [ -f "${WIN64_DIR}/d3d9.dll" ] && [ -f "${WIN64_DIR}/PalDefender.dll" ]; then
        export WINEDLLOVERRIDES="d3d9=n,b"
        echo ">>> 已启用 PalDefender"
    else
        unset WINEDLLOVERRIDES
    fi

    # 启动参数：仅游戏端口和性能参数。
    # REST API 通过 PalWorldSettings.ini 启用，不是命令行参数（传了会导致退出码 3）。
    local args=(-port=8211 -useperfthreads -NoAsyncLoadingThread -UseMultithreadForDS)

    echo ">>> 启动 PalServer (Proton)，参数: ${args[*]}"
    echo ">>> PROTONPATH=${PROTONPATH}"
    # 不用 exec：让守护进程能拿到退出码；proton run 的输出直接透传到 docker logs
    "${PROTONPATH}" run "${EXE}" "${args[@]}"
    local rc=$?
    echo ">>> PalServer 退出，退出码: $rc"
    return $rc
}

# 守护循环：游戏崩溃自动重启，除非用户手动 stop
supervise() {
    # 守护循环必须容忍子进程失败（游戏崩溃返回非零），否则 set -e 会让容器退出
    set +e
    mkdir -p "${RUN_DIR}"
    # 清除可能残留的手动停止标记
    rm -f "${STOP_FILE}"
    local fails=0

    while true; do
        if [ -f "${STOP_FILE}" ]; then
            echo ">>> 检测到停止标记，游戏保持停止"
            # 守护不退出（容器常驻），等待 start 清除标记
            sleep 5
            continue
        fi

        if [ ! -f "${EXE}" ]; then
            echo ">>> 游戏未安装，等待安装..."
            sleep 10
            continue
        fi

        echo ">>> 守护：启动游戏进程"
        launch_game &
        local child=$!
        echo "${child}" > "${PID_FILE}"
        wait "${child}"
        local rc=$?
        rm -f "${PID_FILE}"
        echo ">>> 游戏进程退出（退出码 $rc）"
        # 游戏成功运行一段时间（>60s）才重置失败计数；这里简单处理：
        # 退出码 0 视为正常退出，重置计数；非 0 累加
        if [ $rc -eq 0 ]; then
            fails=0
        fi

        if [ -f "${STOP_FILE}" ]; then
            echo ">>> 手动停止，不自动重启"
            continue
        fi
        # 快速崩溃时退避：失败次数越多等越久，避免狂刷
        fails=$((fails + 1))
        if [ $fails -gt 5 ]; then
            wait_sec=60
        elif [ $fails -gt 3 ]; then
            wait_sec=30
        else
            wait_sec=10
        fi
        echo ">>> ${wait_sec} 秒后自动重启（已连续失败 ${fails} 次）..."
        sleep $wait_sec
    done
}

cmd_start() {
    mkdir -p "${RUN_DIR}"
    rm -f "${STOP_FILE}"
    if [ -f "${PID_FILE}" ] && kill -0 "$(cat "${PID_FILE}")" 2>/dev/null; then
        echo "already running (pid $(cat "${PID_FILE}"))"
        return 0
    fi
    # 通知守护进程拉起（守护在跑的话，清除停止标记后它会自动启动）
    if pgrep -f "entrypoint.sh run" >/dev/null 2>&1 || pgrep -f "supervise" >/dev/null 2>&1; then
        echo ">>> 已通知守护进程启动游戏"
        return 0
    fi
    # 守护没在跑（不应该发生），直接后台启动
    nohup "${0}" run >>"${LOG_FILE}" 2>&1 &
    echo ">>> 已启动守护进程"
}

cmd_stop() {
    mkdir -p "${RUN_DIR}"
    touch "${STOP_FILE}"
    if [ -f "${PID_FILE}" ]; then
        local pid
        pid=$(cat "${PID_FILE}")
        if kill -0 "${pid}" 2>/dev/null; then
            echo ">>> 停止游戏进程 ${pid}..."
            wineserver -k 2>/dev/null || kill -TERM "${pid}" 2>/dev/null || true
            sleep 3
            kill -KILL "${pid}" 2>/dev/null || true
        fi
        rm -f "${PID_FILE}"
    else
        wineserver -k 2>/dev/null || pkill -f PalServer-Win64 2>/dev/null || true
    fi
    echo "stopped"
}

cmd_status() {
    if [ -f "${PID_FILE}" ] && kill -0 "$(cat "${PID_FILE}")" 2>/dev/null; then
        echo "running (pid $(cat "${PID_FILE}"))"
        exit 0
    fi
    # 兜底：pgrep
    if pgrep -f PalServer-Win64 >/dev/null 2>&1; then
        echo "running"
        exit 0
    fi
    echo "stopped"
    exit 1
}

cmd_restart() {
    cmd_stop
    sleep 2
    rm -f "${STOP_FILE}"
    cmd_start
}

# ---- 主流程 ----

# 容器以 root 启动：安装 SteamCMD、修权限，然后降权为 steam 执行
if [ "$(id -u)" = "0" ]; then
    mkdir -p "${STEAMCMD_DIR}" "${PALSERVER_DIR}" /home/steam/prefix "${RUN_DIR}"

    # SteamCMD / Proton 需要这些目录可写。prefix 尤其重要：
    # Wine 要求 prefix 目录属主必须是运行用户，否则报 "is not owned by you"。
    # 先 chown 父目录，再在最后递归 chown，确保 steam 运行中产生的文件也归属正确。
    chown steam:steam /home/steam /home/steam/prefix "${PALSERVER_DIR}" "${STEAMCMD_DIR}" "${RUN_DIR}" 2>/dev/null || true

    if ! install_steamcmd; then
        echo "!!! SteamCMD 安装失败"
        exit 1
    fi

    # 最终递归修正权限（prefix 可能含上次 root 创建的残留）
    chown -R steam:steam "${STEAMCMD_DIR}" "${PALSERVER_DIR}" /home/steam/prefix /home/steam/.steam "${RUN_DIR}" 2>/dev/null || true
    echo ">>> 权限检查：prefix 属主 = $(stat -c '%U:%G' /home/steam/prefix 2>/dev/null || echo unknown)"

    # 降权为 steam 重新执行本脚本。
    # 显式传递 Proton 所需的全部环境变量（su -c 不一定继承 Docker ENV）。
    exec su steam -c '
        STEAMCMD_DIR="'"$STEAMCMD_DIR"'" PALSERVER_DIR="'"$PALSERVER_DIR"'" \
        REST_PORT="'"$REST_PORT"'" REST_PASSWORD="'"$REST_PASSWORD"'" \
        PROTONPATH="'"$PROTONPATH"'" WINEDLLOVERRIDES="'"$WINEDLLOVERRIDES"'" \
        STEAM_COMPAT_CLIENT_INSTALL_PATH="'"${STEAM_COMPAT_CLIENT_INSTALL_PATH:-/home/steam/.steam/root}"'" \
        STEAM_COMPAT_DATA_PATH="'"${STEAM_COMPAT_DATA_PATH:-/home/steam/prefix}"'" \
        STEAMAPPID="'"${STEAMAPPID:-2394010}"'" XDG_RUNTIME_DIR=/tmp/runtime-steam \
        _ENTRY_ARG="'"${1:-run}"'" bash "'"$0"'"
    '
fi

# ---- 以下以 steam 用户运行 ----
export XDG_RUNTIME_DIR=/tmp/runtime-steam
mkdir -p "${XDG_RUNTIME_DIR}" 2>/dev/null || true

case "${_ENTRY_ARG:-run}" in
    run)
        # 容器常驻：守护游戏进程
        supervise
        ;;
    start)
        cmd_start
        ;;
    stop)
        cmd_stop
        ;;
    restart)
        cmd_restart
        ;;
    status)
        cmd_status
        ;;
    install|update)
        install_server
        ;;
    *)
        exec "$@"
        ;;
esac
