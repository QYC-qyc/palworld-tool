# Palworld 游戏服镜像：SteamCMD + GE-Proton + PalServer + PalDefender
# 在 Linux 上通过 Proton 运行 Windows 版 PalServer.exe，以加载 PalDefender 反作弊 DLL。
FROM debian:bookworm-slim

# 避免安装交互
ENV DEBIAN_FRONTEND=noninteractive \
    STEAMAPPID=2394010 \
    PALSERVER_DIR=/home/steam/palserver \
    PROTON_VERSION=9-22

# 安装系统依赖：
# - wget/curl/xz-utils/cabextract：下载和解压 GE-Proton
# - ca-certificates：HTTPS
# - 32/64 位库：Proton 运行 Windows 程序所需
# - dbus-x11：Proton 初始化需要
RUN dpkg --add-architecture i386 && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates wget curl xz-utils cabextract \
        lib32gcc-s1 lib32stdc++6 \
        libc6:i386 libstdc++6:i386 libgcc-s1:i386 \
        dbus-x11 locales \
        && \
    sed -i 's/^# *\(en_US.UTF-8\)/\1/' /etc/locale.gen && \
    locale-gen en_US.UTF-8 && \
    # 手动安装 SteamCMD（Debian 官方源的 steamcmd 包在 bookworm 上依赖问题多）
    mkdir -p /opt/steamcmd && \
    cd /opt/steamcmd && \
    wget -q "https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz" -O steamcmd.tar.gz && \
    tar -xzf steamcmd.tar.gz && \
    rm steamcmd.tar.gz && \
    ln -sf /opt/steamcmd/steamcmd.sh /usr/local/bin/steamcmd && \
    ln -sf /opt/steamcmd/linux32/ /opt/steamcmd/linux64/ && \
    # 首次运行 steamcmd 完成自更新
    steamcmd +login anonymous +quit || true && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# 创建非 root 用户
RUN useradd -m -d /home/steam -s /bin/bash steam

# 下载并安装 GE-Proton（经国内镜像加速，失败重试）
WORKDIR /home/steam
RUN mkdir -p /home/steam/.steam/root/compatibilitytools.d && \
    cd /home/steam/.steam/root/compatibilitytools.d && \
    echo "下载 GE-Proton ${PROTON_VERSION}..." && \
    (wget -q --tries=3 --timeout=120 \
      "https://ghfast.top/https://github.com/GloriousEggroll/proton-ge-custom/releases/download/GE-Proton${PROTON_VERSION}/GE-Proton${PROTON_VERSION}.tar.gz" -O proton.tar.gz || \
     wget -q --tries=3 --timeout=120 \
      "https://github.com/GloriousEggroll/proton-ge-custom/releases/download/GE-Proton${PROTON_VERSION}/GE-Proton${PROTON_VERSION}.tar.gz" -O proton.tar.gz) && \
    ls -lh proton.tar.gz && \
    tar -xzf proton.tar.gz && rm proton.tar.gz && \
    chown -R steam:steam /home/steam

ENV STEAM_COMPAT_CLIENT_INSTALL_PATH=/home/steam/.steam/root \
    STEAM_COMPAT_DATA_PATH=/home/steam/prefix \
    PROTONPATH=/home/steam/.steam/root/compatibilitytools.d/GE-Proton${PROTON_VERSION}/proton \
    WINEDLLOVERRIDES="d3d9=n,b" \
    LANG=en_US.UTF-8 \
    LC_ALL=en_US.UTF-8

# 复制启动脚本
COPY --chmod=755 docker/gameserver/entrypoint.sh /home/steam/entrypoint.sh

# 游戏服端口
EXPOSE 8211/udp 8212/tcp

# 游戏数据持久化
VOLUME ["/home/steam/palserver"]

USER steam
WORKDIR /home/steam

# 默认启动游戏服
ENTRYPOINT ["/home/steam/entrypoint.sh"]
CMD ["start"]
