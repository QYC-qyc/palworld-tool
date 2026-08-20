# Palworld 游戏服镜像：SteamCMD + GE-Proton + PalServer + PalDefender
# 在 Linux 上通过 Proton 运行 Windows 版 PalServer.exe，以加载 PalDefender 反作弊 DLL。
FROM debian:bookworm-slim

# 避免安装时交互弹窗
ENV DEBIAN_FRONTEND=noninteractive \
    STEAMAPPID=2394010 \
    PALSERVER_DIR=/home/steam/palserver \
    PROTON_VERSION=GE-Proton9-21

# 安装系统依赖：
# - steamcmd 下载游戏服
# - 32/64 位库供 Proton 运行 Windows 程序
# - wget/cabextract 用于安装 Proton
# - ca-certificates HTTPS
RUN dpkg --add-architecture i386 && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates wget curl cabextract xz-utils gpg \
        lib32gcc-s1 lib32stdc++6 \
        libc6:i386 libstdc++6:i386 libgcc-s1:i386 \
        dbus-x11 \
        && \
    # SteamCMD 官方源
    echo "deb http://deb.debian.org/debian bookworm contrib non-free non-free-firmware" > /etc/apt/sources.list.d/contrib.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends steamcmd && \
    ln -s /usr/games/steamcmd /usr/bin/steamcmd && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# 创建非 root 用户
RUN useradd -m -d /home/steam -s /bin/bash steam

# 下载并安装 GE-Proton
WORKDIR /home/steam
RUN mkdir -p /home/steam/.steam/root/compatibilitytools.d && \
    cd /home/steam/.steam/root/compatibilitytools.d && \
    echo "downloading GE-Proton ${PROTON_VERSION}..." && \
    wget -q "https://github.com/GloriousEggroll/proton-ge-custom/releases/download/${PROTON_VERSION}/${PROTON_VERSION}.tar.gz" -O proton.tar.gz && \
    tar -xzf proton.tar.gz && rm proton.tar.gz && \
    chown -R steam:steam /home/steam

ENV STEAM_COMPAT_CLIENT_INSTALL_PATH=/home/steam/.steam/root \
    STEAM_COMPAT_DATA_PATH=/home/steam/prefix \
    PROTONPATH=/home/steam/.steam/root/compatibilitytools.d/${PROTON_VERSION}/proton \
    WINEDLLOVERRIDES="d3d9=n,b"

# 复制启动脚本
COPY --chmod=755 entrypoint.sh /home/steam/entrypoint.sh
COPY --chown=steam:steam install.sh /home/steam/install.sh
RUN chmod +x /home/steam/install.sh

# 游戏服端口：游戏 8211/udp，REST 8212/tcp
EXPOSE 8211/udp 8212/tcp

# 游戏数据持久化卷
VOLUME ["/home/steam/palserver"]

USER steam
WORKDIR /home/steam

# 默认启动游戏服
ENTRYPOINT ["/home/steam/entrypoint.sh"]
CMD ["start"]
