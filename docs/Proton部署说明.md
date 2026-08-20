# Proton 模式部署说明

面板只运行 **Windows 版游戏服**，通过 [GE-Proton](https://github.com/GloriousEggroll/proton-ge-custom) 在 Linux 上运行，目的是加载 PalDefender（Windows DLL 反作弊）。

## 组件关系

```
PalAdmin (Go 二进制, root)
  ├── SteamCMD (force platform windows)
  │     └─ 下载 Windows 版游戏服 → <安装目录>/PalServer-Win/
  ├── GE-Proton (/opt/GE-Proton/proton)
  │     └─ proton run PalServer-Win64-Shipping-Cmd.exe
  ├── PalDefender DLL → Pal/Binaries/Win64/ (d3d9.dll + PalDefender.dll)
  └── REST API → 游戏服 :8212
```

## 关键路径

| 项目 | 路径 |
|---|---|
| 用户填写的安装目录 | 如 `/home/paladmin/PalServer` |
| Windows 版游戏服 | `<安装目录>/PalServer-Win/` |
| 游戏可执行文件 | `PalServer-Win/Pal/Binaries/Win64/PalServer-Win64-Shipping-Cmd.exe` |
| 配置文件 | `PalServer-Win/Pal/Saved/Config/WindowsServer/PalWorldSettings.ini` |
| 存档 | `PalServer-Win/Pal/Saved/SaveGames/` |
| PalDefender DLL | `PalServer-Win/Pal/Binaries/Win64/{d3d9.dll,PalDefender.dll}` |
| PalDefender 配置 | `PalServer-Win/Pal/Binaries/Win64/PalDefender/Config.json`、`Banlist.json`、`Tokens/` |
| GE-Proton | `/opt/GE-Proton/`（proton 脚本在根目录） |
| Proton prefix | `<安装目录>/PalServer-Win/proton_prefix/` |

## Proton 启动环境变量

`internal/gamesrv/manager.go` 启动游戏服时设置：

- `PROTON_DIST_PATH=<proton 所在目录>`（GE-Proton 解压根，含 dist/ 和 filelock/）
- `PROTON_NO_STEAM=1` — 无 Steam 客户端运行
- `PROTON_NO_ESYNC=1`
- `STEAM_COMPAT_CLIENT_INSTALL_PATH=<SteamCMD 目录>`
- `STEAM_COMPAT_DATA_PATH=<PalServer-Win>/proton_prefix`
- `WINEDLLOVERRIDES=d3d9=n,b` — 加载 PalDefender 的 d3d9.dll

若使用系统包管理器装的 proton（dist 在别处），在「设置 → Proton 路径」手动指定可执行文件路径。

## 一键安装 Proton 做了什么

`api/paldefender.go` 的 `installProton`（后台任务，进度通过 `/paldefender/proton-status` 轮询）：

1. **检测系统**：读 `/etc/os-release`，仅支持 Debian/Ubuntu（apt）和 Arch（pacman）；其他系统提示手动安装。
2. **装系统依赖**：
   - Debian/Ubuntu：`dpkg --add-architecture i386` → `apt-get update` → 装 curl/tar/xz-utils + i386 库
     （libc6:i386、libstdc++6:i386、libgcc-s1:i386、lib32gcc-s1、lib32stdc++6）
   - Arch：`pacman -Sy --needed curl tar xz wine lib32-gcc-libs lib32-glibc`
3. **下载 GE-Proton**：从 GitHub API 取最新 release 的 tar.gz，经镜像（ghfast.top/gh-proxy.com/ghproxy.net/直连）下载到临时文件。
4. **解压**：`tar -xzf --strip-components=1 -C /opt/GE-Proton`（先清空旧版）。
5. **验证**：检查 `/opt/GE-Proton/proton` 存在并 chmod +x。

### apt-get exit 100

常见原因：
- 包名不存在（`steam-libs-i386` 是 Arch 包名，Debian 系勿用）——已修复为发行版对应的 i386 库包名
- apt 数据库锁（另一 apt 进程在跑）——等待后重试
- 软件源不可达——检查网络/DNS，或手动 `apt-get update`
- i386 架构未启用——确认 `dpkg --print-foreign-architectures` 含 i386

设置了 `DEBIAN_FRONTEND=noninteractive` 等避免交互卡住。安装后会跑 `apt-get -f install` 尝试修复破损依赖。

## 启动前校验

点击「启动」时，`checkProtonReady()` 逐项检查，缺任何一项都直接返回错误（不回退原生）：

1. Proton 可执行文件存在（`protonExePath()`：设置路径 > /opt/GE-Proton/proton > /usr/bin/proton > glob 用户目录）
2. Windows 版游戏服 exe 存在（`winServerExePath()`）
3. PalDefender DLL 存在（d3d9.dll + PalDefender.dll）

错误信息会说明缺什么、去哪装。

## 进程检测与停止

- 进程名匹配：`pgrep/pkill -f PalServer-Win64-Shipping-Cmd.exe`（Proton 下进程名仍是 exe 名）
- 兜底：`pkill -f "proton.*PalServer"`
- **不使用** wineserver -k（Proton 不用 wineserver 机制）

## Windows 版下载

`Install()` 无参数，始终强制下载 Windows 版：
```
steamcmd.sh +force_install_dir <安装目录>/PalServer-Win +login anonymous \
  +@sSteamCmdForcePlatformType windows +@sSteamCmdForcePlatformBitness 64 \
  +app_update 2394010 validate +quit
```

首次失败（Disk write failure）会自动清 `appcache/depotcache/logs` 缓存重试一次。Linux 版和 Windows 版装在不同子目录，互不覆盖。

## 与旧版（Linux 原生）的区别

- 不再有 Linux/Windows 模式切换开关，永远是 Windows 版 + Proton
- 配置文件固定 `WindowsServer/PalWorldSettings.ini`（之前按模式选 LinuxServer/WindowsServer）
- 玩家踢/封/解封走 PalDefender REST API（`/paldefender/api/...`），不再用官方 REST 兜底
- 存档路径指向 `PalServer-Win/Pal/Saved`，备份/解码用此路径
