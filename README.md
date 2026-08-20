# PalAdmin

幻兽帕鲁（Palworld）服务器管理与反作弊面板。

- 🪟 **Windows 版游戏服**：通过 **Proton** 在 Linux 上运行 Windows 版游戏服，以加载 PalDefender（进程级反作弊 DLL）
- 🛡️ **PalDefender 反作弊**：实时拦截属性修改、非法物品、违禁科技等作弊，支持踢出/封禁/IP 封禁；面板内一键安装 Proton 与 PalDefender
- 🎮 **SteamCMD 管理**：面板内一键安装/更新 Windows 版服务端，启动、停止、重启，无需手写命令
- 🗺️ **交互地图**：Leaflet 世界地图（主世界 / 天坠之地），实时显示在线玩家、据点、Boss 塔、快速旅行点
- 👥 **玩家管理**：在线/存档玩家列表，查看背包物品（WebP 图标）、帕鲁详情，踢/封/解封
- 🏛️ **公会**：公会信息与成员列表
- 💾 **存档备份与回档**：定时备份，一键回滚（自动停服、备份、恢复、启服）
- ⚙️ **可视化配置**：`PalWorldSettings.ini` 全部参数按分类编辑，字段含中文说明和取值范围；自动按 Proton/Windows 版定位 `WindowsServer/PalWorldSettings.ini`
- 🌗 **暗色主题**、响应式、手机端适配

## 架构

```
浏览器（Vue3 + Naive UI）
        │  http://服务器IP:8190
        ▼
PalAdmin（宿主机二进制，root 运行）
        ├── SteamCMD  →  下载/更新 Windows 版游戏服（App 2394010）
        ├── Proton    →  运行 PalServer-Win64-Shipping-Cmd.exe
        │                 （STEAM_COMPAT_DATA_PATH=<安装目录>/PalServer-Win/proton_prefix）
        ├── PalDefender DLL 注入游戏进程（d3d9=n,b）
        └── REST API  →  连接游戏服 :8212 同步玩家/执行操作
```

**关键路径：**
- 游戏安装目录：用户在面板填写（如 `/home/paladmin/PalServer`）
- Windows 版实际安装在：`<安装目录>/PalServer-Win/`（与可能存在的 Linux 版隔离，避免文件互相覆盖）
- 游戏可执行文件：`PalServer-Win/Pal/Binaries/Win64/PalServer-Win64-Shipping-Cmd.exe`
- 配置文件：`PalServer-Win/Pal/Saved/Config/WindowsServer/PalWorldSettings.ini`
- 存档：`PalServer-Win/Pal/Saved/SaveGames/`
- PalDefender：`PalServer-Win/Pal/Binaries/Win64/`（d3d9.dll / PalDefender.dll）
- GE-Proton：一键安装到 `/opt/GE-Proton/`

## 部署

### 前置条件

- Linux x86_64 / arm64 服务器（推荐 Ubuntu 22.04 / Debian 12）
- 开放端口：面板 `8190/tcp`、游戏 `8211/udp`、REST `8212/tcp`（REST 建议仅内网）

### 一键安装（推荐）

```bash
curl -fsSL https://gitee.com/QYC-qyc/palworld-tool/raw/main/scripts/install.sh | sudo bash
```

脚本自动从 GitHub Release（经多镜像测速选最快源）下载对应架构二进制，安装到 `/opt/paladmin/`，数据存 `/var/lib/paladmin/`，注册 systemd 服务。

```bash
systemctl status paladmin      # active (running)
journalctl -u paladmin -f      # 日志
```

访问 `http://服务器IP:8190`，首次设置面板登录密码。

**更新：** 重跑安装脚本即可（配置与数据保留），然后 `systemctl restart paladmin`。

### Docker

```bash
mkdir -p /www/palworld-tool && cd /www/palworld-tool
curl -o docker-compose.yml \
  https://gitee.com/QYC-qyc/palworld-tool/raw/main/docker-compose.yml
docker compose up -d
```

> 注意：Proton 运行 Windows 游戏服建议用**二进制直装**（宿主机直接管理进程）。Docker 方式下 Proton/Systemd 模式支持有限。

## 使用流程

1. **设置面板密码**（首次访问）
2. **「游戏服」页**填写：
   - SteamCMD 目录：如 `/home/paladmin/steamcmd`
   - 游戏安装目录：如 `/home/paladmin/PalServer`
   - 保存配置
3. **安装 SteamCMD**：在「游戏服」页 SteamCMD 目录旁点击「**安装**」按钮，面板会自动下载 steamcmd_linux.tar.gz、解压并完成首次自更新。
   - 也可手动安装：
     ```bash
     sudo mkdir -p /home/paladmin/steamcmd && cd /home/paladmin/steamcmd
     curl -sqL https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz | sudo tar zxv
     ./steamcmd.sh +login anonymous +quit   # 首次运行自更新
     ```
4. **「PalDefender」页 → 一键安装 Proton**（自动检测系统、装 i386 依赖、经镜像下载最新 GE-Proton 到 /opt/GE-Proton；ARM64 服务器还会自动装 box64）
5. **「游戏服」页 → 安装/更新游戏服**（SteamCMD 强制下载 Windows 版到 `PalServer-Win/` 子目录）
6. **「PalDefender」页 → 安装 PalDefender**（下载 DLL 到 Win64 目录）；在「设置」页可配置反作弊开关（踢出/封禁/IP封禁），保存时写入 PalDefender/Config.json 并热重载
7. **「游戏服」页 → 启动**（通过 `proton run` 启动，自动注入 PalDefender）
8. **「游戏配置」页**开启 REST API：`RESTAPIEnabled=true`、`RESTAPIPort=8212`、设置 `AdminPassword`，保存并重启
9. 完成后仪表盘/地图/玩家页即可同步数据

> 启动前会校验 Proton、Windows exe、PalDefender DLL 三项（PalDefender DLL 缺失仅警告不阻止启动）；缺任何必要项都会明确报错，不会静默失败。

### 防火墙

```bash
ufw allow 8190/tcp && ufw allow 8211/udp
```

云服务器安全组同样放行。

## 功能一览

| 功能 | 说明 |
|---|---|
| 仪表盘 | 服务器名/版本/在线/FPS 统计卡、运行模式、控制台（广播/同步/关服） |
| 游戏服 | 一键安装 SteamCMD、SteamCMD 与安装目录配置、安装/更新 Windows 版、启停重启、实时日志（`\r` 进度实时显示）、磁盘空间检查 |
| PalDefender | 一键安装 GE-Proton（按 OS 选包）、安装/卸载 PalDefender DLL、生成 Token；玩家管理、封禁列表、广播警报、公会据点、配置热重载 |
| 游戏配置 | 可视化编辑 PalWorldSettings.ini，参数分类带说明，保存即写 WindowsServer 路径 |
| 玩家 | 在线/存档玩家 Tab 切换、搜索、详情（基本信息/帕鲁/物品，物品用 WebP 图标）、踢/封/解封/IP封禁（走 PalDefender） |
| 玩家地图 | Leaflet 地图，玩家/据点/Boss塔/快速旅行点，主世界/天坠之地切换 |
| 公会 | 公会与成员列表 |
| 白名单 | 白名单管理，非白名单玩家自动踢出 |
| 备份 | 存档备份列表、一键回档；cron 定时备份（游戏运行时跳过） |
| 审计 | 操作日志记录与筛选 |
| 设置 | 面板密码、REST 连接、PalDefender API、反作弊处置开关、存档路径、进程模式(systemd/docker/noop)、面板自更新 |

## 运维命令

```bash
systemctl status paladmin     # 状态
systemctl restart paladmin    # 重启
journalctl -u paladmin -f     # 日志
```

游戏服的安装/启停统一在面板操作。Proton 日志、SteamCMD 下载进度均在面板「游戏服」「PalDefender」页实时查看。

## 自更新

面板「设置」页可检查更新并一键升级；后端会先对多个 GitHub 镜像测速，选最快的下载二进制。也可重跑 install.sh。

## 开发

- **后端**：Go，入口 `main.go`；路由 `api/router.go`；游戏服管理 `internal/gamesrv/`；PalDefender 代理 `internal/paldefender/` + `api/paldefender*.go`；存档解析 `internal/tool/`（参考 palworld-save-tools）
- **前端**：Vue3 + TypeScript + Vite + Naive UI，代码在 `web/`
- **详细约定**：见 [docs/开发约定.md](docs/开发约定.md)
- **构建产物**：GitHub Actions 打 amd64/arm64 二进制，前端 `web/dist` 经 `embed` 打包进单一二进制

构建：
```bash
cd web && npm install && npm run build && cd ..
go build -o paladmin
```

## 致谢

- [palworld-server-tool](https://github.com/zaigie/palworld-server-tool) — 存档解析与管理框架参考（`_palworld-server-tool/`）
- [PalDefender](https://github.com/Ultimeit/PalDefender) — 反作弊
- [palworld-save-tools](https://github.com/cheahjs/palworld-save-tools) — 存档格式解析
- [GE-Proton](https://github.com/GloriousEggroll/proton-ge-custom) — Windows 兼容层
- [LootLab](https://lootlab.cn/palworld) — 配置参数范围、图标、地图数据

## 许可证

MIT
