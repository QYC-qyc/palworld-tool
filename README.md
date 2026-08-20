# PalAdmin

幻兽帕鲁（Palworld）服务器管理与反作弊面板，支持 **Docker 一键部署**（Linux + GE-Proton 运行 Windows 版游戏服 + PalDefender 反作弊）。

- 🐳 **Docker 化部署**：`docker compose up -d` 一键启动面板 + 游戏服，自动挂载数据卷
- 🛡️ **PalDefender 反作弊**：通过 GE-Proton 加载 Windows DLL，实时拦截属性修改、非法物品、违禁科技，支持踢出/封禁/IP 封禁
- 🎮 **SteamCMD 管理**：面板内一键安装/更新 Windows 版服务端，启动、停止、重启，查看实时日志
- 🗺️ **交互地图**：Leaflet 世界地图（主世界 / 天坠之地），实时显示在线玩家、据点、Boss 塔、快速旅行点
- 👥 **玩家管理**：在线/存档玩家列表，查看背包物品（WebP 图标）、帕鲁详情，踢/封/解封（走 PalDefender）
- 🏛️ **公会**：公会信息与成员列表
- 💾 **存档备份与回档**：定时备份，一键回滚（自动停服、备份、恢复、启服）
- ⚙️ **可视化配置**：`PalWorldSettings.ini` 全部参数按分类编辑，字段含中文说明和取值范围
- 🌗 **暗色主题**、响应式、手机端适配
- 🔄 **在线自更新**：经国内镜像测速下载，一键升级

## 架构

```
浏览器（Vue3 + Naive UI）
        │  http://服务器IP:8190
        ▼
┌─ paladmin 容器 ────────┐    REST :8212     ┌─ gameserver 容器 ───────┐
│ Go + Vue 面板          │ ───────────────► │ SteamCMD + GE-Proton    │
│ :8190                  │                  │ PalServer-Win64.exe     │
│ 挂载 /var/run/docker.sock ── docker CLI ──► │ + PalDefender DLL       │
└────────────────────────┘                  └─────────────────────────┘
        │                                          │
        └──────── 卷：./data/paladmin              └─ 卷：./data/gameserver
```

**两容器通过 Docker 网络互联：**
- **gameserver**：内含 SteamCMD + GE-Proton + PalServer + PalDefender，对外暴露 8211/udp、8212/tcp
- **paladmin**：面板，通过挂载 `docker.sock` 用 Docker CLI 管控游戏服容器的启停/日志/安装
- 面板通过 `http://gameserver:8212` 连接游戏服 REST API
- 游戏数据持久化到宿主机 `./data/gameserver`，面板数据到 `./data/paladmin`

## 快速开始（Docker 推荐）

### 前置条件
- Linux x86_64 服务器（推荐 Ubuntu 22.04 / Debian 12）
- 已安装 Docker 和 Docker Compose Plugin：
  ```bash
  curl -fsSL https://get.docker.com | sh
  ```
- 至少 5GB 磁盘、2GB+ 内存
- 开放端口：8190/tcp（面板）、8211/udp（游戏）、8212/tcp（REST，建议仅内网）

### 启动（纯镜像，无需克隆代码）

```bash
mkdir -p ~/paladmin && cd ~/paladmin
curl -fsSL -o docker-compose.yml \
  https://gitee.com/qyc-qyc/palworld-tool/raw/main/docker-compose.prod.yml
docker compose up -d
docker compose ps
```

访问 `http://服务器IP:8190`，首次设置面板密码。

> 若拉取镜像报 401，需在 GitHub 上将 `paladmin`、`palworld-gameserver` 两个 package 设为 Public（详见 [docs/Docker部署.md](docs/Docker部署.md)）。

<details>
<summary>从源码构建</summary>

```bash
git clone <repo-url> paladmin && cd paladmin
docker compose build
docker compose up -d
```

</details>

### 首次使用流程

1. **设置密码**（首次访问）
2. **安装游戏服**：进入「游戏服」页，确认游戏安装目录为容器内 `/home/steam/palserver`（已映射到 `./data/gameserver`），点击「安装/更新游戏服」
3. **安装 PalDefender**：进入「PalDefender」页，点击「安装 PalDefender」，DLL 会下载到游戏服 Win64 目录
4. **启动游戏服**：点击「启动」（面板执行 `docker start palworld-gameserver`）
5. **开启 REST API**：在「游戏配置」页设置 `RESTAPIEnabled=true`、`RESTAPIPort=8212`、`AdminPassword=...`，保存并重启
6. 完成后仪表盘/地图/玩家页即可同步数据

详细说明见 [docs/Docker部署.md](docs/Docker部署.md)。

### 非 Docker / 二进制直装

仍支持在 Linux 上直接运行二进制（宿主机装 SteamCMD + GE-Proton）：

```bash
curl -fsSL https://gitee.com/QYC-qyc/palworld-tool/raw/main/scripts/install.sh | sudo bash
```

详见 [docs/开发约定.md](docs/开发约定.md) 中"非容器部署"。二进制包在 [GitHub Release](https://github.com/QYC-qyc/palworld-tool/releases)。

## 功能一览

| 功能 | 说明 |
|---|---|
| 仪表盘 | 服务器名/版本/在线/FPS 统计、控制台（广播/同步/关服） |
| 游戏服 | Docker 容器启停、安装/更新 Windows 版、实时日志、磁盘空间检查 |
| PalDefender | 安装/卸载 PalDefender DLL、生成 Token、玩家管理、封禁列表、广播警报、公会据点、配置热重载 |
| 游戏配置 | 可视化编辑 PalWorldSettings.ini，参数分类带说明，保存即写 WindowsServer 路径 |
| 玩家 | 在线/存档玩家 Tab、搜索、详情（基本信息/帕鲁/物品，物品用 WebP 图标）、踢/封/解封/IP封禁 |
| 玩家地图 | Leaflet 地图，玩家/据点/Boss塔/快速旅行点，主世界/天坠之地切换 |
| 公会 | 公会与成员列表 |
| 白名单 | 白名单管理，非白名单自动踢出 |
| 备份 | 存档备份列表、一键回档；cron 定时备份（游戏运行时跳过） |
| 审计 | 操作日志 |
| 设置 | 面板密码、REST 连接、PalDefender API、反作弊处置开关、存档路径、进程模式、面板自更新 |

## 运维命令

```bash
docker compose logs -f paladmin       # 面板日志
docker compose logs -f gameserver     # 游戏服日志（也可在面板看）
docker compose restart paladmin       # 重启面板
docker compose down                   # 停止全部
docker compose exec gameserver bash   # 进入游戏服容器排查

# 手动更新游戏服
docker compose exec gameserver /home/steam/entrypoint.sh update

# 备份游戏存档
tar -czf backup-$(date +%F).tar.gz ./data/gameserver/SaveGames
```

## 目录结构

```
.
├── api/                  # Gin 路由与 handler
├── internal/
│   ├── gamesrv/          # 游戏服进程/Docker 容器管理
│   ├── github/           # GitHub 国内镜像统一封装
│   ├── paldefender/      # PalDefender REST 客户端
│   ├── tool/             # 存档解析、REST API
│   └── ...
├── web/                  # Vue3 + TypeScript + Vite 前端
├── docker/
│   ├── gameserver/       # 游戏服镜像（SteamCMD+Proton+PalServer）
│   └── paladmin/         # 面板镜像（多阶段构建）
├── docker-compose.yml    # 编排两容器
├── docs/                 # 文档
└── scripts/              # 安装脚本等
```

## 开发

- **后端**：Go 1.22，入口 `main.go`；路由 `api/router.go`；游戏服管理 `internal/gamesrv/`
- **前端**：Vue3 + TypeScript + Vite + Naive UI，代码在 `web/`
- **Docker**：游戏服镜像 `docker/gameserver/`，面板镜像 `docker/paladmin/`
- **详细约定**：见 [docs/开发约定.md](docs/开发约定.md)
- **Docker 部署**：见 [docs/Docker部署.md](docs/Docker部署.md)

本地构建：
```bash
# 后端
go build -o paladmin .
# 前端
cd web && npm install && npm run build
# Docker 镜像
docker compose build
```

## 致谢

- [palworld-server-tool](https://github.com/zaigie/palworld-server-tool) — 存档解析与管理框架参考
- [PalDefender](https://github.com/Ultimeit/PalDefender) — 反作弊
- [palworld-save-tools](https://github.com/cheahjs/palworld-save-tools) — 存档格式解析
- [GE-Proton](https://github.com/GloriousEggroll/proton-ge-custom) — Windows 兼容层
- [LootLab](https://lootlab.cn/palworld) — 配置参数范围、图标、地图数据

## 许可证

MIT
