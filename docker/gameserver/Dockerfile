# Palworld 游戏服镜像：SteamCMD + GE-Proton + PalServer + PalDefender
# 在 Linux 上通过 Proton 运行 Windows 版 PalServer.exe，以加载 PalDefender 反作弊 DLL。
FROM debian:bookworm-slim

# 避免安装交互
ENV DEBIAN_FRONTEND=noninteractive \
    STEAMAPPID=2394010 \
    PALSERVER_DIR=/home/steam/palserver \
    PROTON_VERSION=9-22

# 换阿里云 apt 镜像（国内构建加速）。用 http，避免 slim 镜像缺 ca-certificates 导致 https 失败。
RUN set -eux; \
    if [ -f /etc/apt/sources.list.d/debian.sources ]; then \
        sed -i -E 's|https?://deb.debian.org|http://mirrors.aliyun.com|g; s|https?://security.debian.org|http://mirrors.aliyun.com|g' /etc/apt/sources.list.d/debian.sources; \
    fi; \
    if [ -f /etc/apt/sources.list ]; then \
        sed -i -E 's|https?://deb.debian.org|http://mirrors.aliyun.com|g; s|https?://security.debian.org|http://mirrors.aliyun.com|g' /etc/apt/sources.list; \
    fi

# 安装系统依赖：
# - wget/curl/xz-utils/cabextract：下载和解压 GE-Proton
# - ca-certificates：HTTPS
# - 32/64 位库：Proton 运行 Windows 程序所需
# - dbus-x11：Proton 初始化需要
RUN dpkg --add-architecture i386 && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates wget curl xz-utils cabextract \
        lib32gcc-s1 lib32stdc++6 procps \
        libc6:i386 libstdc++6:i386 libgcc-s1:i386 \
        dbus-x11 locales \
        && \
    sed -i 's/^# *\(en_US.UTF-8\)/\1/' /etc/locale.gen && \
    locale-gen en_US.UTF-8 && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# 创建非 root 用户
RUN useradd -m -d /home/steam -s /bin/bash steam

# SteamCMD 在容器首次启动时由 entrypoint 下载安装到可写卷 /opt/steamcmd，
# 这样：1) 自更新成功一次后持久保留，容器重启不重新联网；2) 镜像更小。
# 预创建目录并赋权给 steam 用户（运行时用户）。
RUN mkdir -p /opt/steamcmd /home/steam/palserver && chown -R steam:steam /opt/steamcmd /home/steam

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
    echo "清理 GE-Proton 冗余（游戏服作为 64 位 Windows 服务端，不需要 Mono/Gecko/文档/字体）..." && \
    PROTODIR="GE-Proton${PROTON_VERSION}" && \
    # Mono（.NET）和 Gecko（IE 控件）——游戏服完全不需要，通常各占 50-100MB
    find "$PROTODIR" -maxdepth 6 -type d \( -iname "mono" -o -iname "gecko" \) -prune -exec rm -rf {} + 2>/dev/null || true && \
    # 文档、man、locale、示例
    rm -rf "$PROTODIR/files/share"/{man,doc,documentation,info,lintian,examples,bash-completion,zsh,applications} 2>/dev/null || true && \
    find "$PROTODIR/files/share" -type d \( -name "locale" -o -name "i18n" -o -name "man*" -o -name "doc" \) -prune -exec rm -rf {} + 2>/dev/null || true && \
    # 字体（游戏服不渲染界面）
    rm -rf "$PROTODIR/files/share/wine/fonts" "$PROTODIR/files/share/fonts" 2>/dev/null || true && \
    # 杂项文档/许可文件（保留目录结构，只删文件）
    find "$PROTODIR" -type f \( -name "*.html" -o -name "*.md" -o -name "ANNOUNCE" -o -name "README*" -o -name "NEWS*" -o -name "AUTHORS*" \) -delete 2>/dev/null || true && \
    find "$PROTODIR" -type d -name "__pycache__" -prune -exec rm -rf {} + 2>/dev/null || true && \
    echo "清理后 Proton 大小：" && du -sh "$PROTODIR" && \
    chown -R steam:steam /home/steam

# Proton 运行时数据（Wine prefix）独立挂载持久化，避免容器重建后 Proton 重新初始化。
# 注意：PalServer 存档始终写在游戏目录 Pal/Saved 下，与 prefix 位置无关。
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

# 数据持久化：游戏目录 + SteamCMD（均由 compose 映射到宿主机）
VOLUME ["/home/steam/palserver", "/opt/steamcmd"]

# 容器以 root 启动（处理挂载卷权限、安装 SteamCMD），
# 启动游戏服时由 entrypoint 降权给 steam 用户。
USER root
WORKDIR /home/steam

# 默认启动游戏服
ENTRYPOINT ["/home/steam/entrypoint.sh"]
CMD ["start"]
