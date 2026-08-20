# Docker 部署指南（Linux + Proton + PalDefender）

本项目提供 Docker Compose 编排，一键启动 PalAdmin 面板 + PalServer 游戏服（通过 GE-Proton 运行 Windows 版并加载 PalDefender 反作弊）。

## 架构

```
┌─ Docker Host (Linux) ─────────────────────────────┐
│                                                  │
│  ┌─ paladmin 容器 ────┐    REST :8212           │
│  │ Go + Vue 面板      │ ───────────────────────► │
│  │ :8190              │                          │
│  │ 挂载 docker.sock   │ ── docker CLI ──►        │
│  └────────────────────┘                  ┌─ gameserver 容器 ─┐
│                                          │ SteamCMD+Proton   │
│                                          │ PalServer.exe     │
│  卷:                                     │ +PalDefender DLL  │
│  ./data/paladmin    (面板数据库)          └───────────────────┘
│  ./data/gameserver  (游戏存档/配置)              │
└──────────────────────────────────────────────────┘
```

- **gameserver 容器**：内含 SteamCMD + GE-Proton + PalServer + PalDefender，对外暴露 8211/udp、8212/tcp
- **paladmin 容器**：面板，通过挂载 `/var/run/docker.sock` 用 Docker CLI 管控 gameserver 容器的启停/日志/安装
- 两容器在同一 Docker 网络，面板通过 `http://gameserver:8212` 连游戏服 REST API

## 前置条件

- Linux x86_64 服务器（推荐 Ubuntu 22.04 / Debian 12）
- 已安装 Docker 和 Docker Compose Plugin
  ```bash
  curl -fsSL https://get.docker.com | sh
  ```
- 至少 5GB 磁盘空间、2GB+ 内存
- 开放端口：8190/tcp（面板）、8211/udp（游戏）、8212/tcp（REST，建议仅内网）

## 快速开始

```bash
# 1. 克隆项目
git clone <repo-url> paladmin && cd paladmin

# 2. 构建并启动
docker compose build
docker compose up -d

# 3. 查看状态
docker compose ps
docker compose logs -f paladmin
```

访问 `http://服务器IP:8190`，首次设置面板密码。

### 首次安装游戏服

1. 在「游戏服」页填写：
   - SteamCMD 目录：`/usr/games`（容器内已装 steamcmd，也可填任意目录点「安装」）
   - 游戏安装目录：`/home/steam/palserver`（这是容器内路径，已映射到宿主机 `./data/gameserver`）
2. 点击「安装/更新游戏服」——面板会执行 `docker exec palworld-gameserver /home/steam/entrypoint.sh update`
3. 安装完成后点击「启动」——面板执行 `docker start palworld-gameserver`
4. 在「PalDefender」页安装 PalDefender DLL（会下载到 gameserver 容器的 Win64 目录）
5. 在「游戏配置」页开启 REST API（`RESTAPIEnabled=true`、`RESTAPIPort=8212`、设置 `AdminPassword`），保存并重启游戏服

> 注意：由于是容器化部署，面板的"安装 SteamCMD"和"一键安装 Proton"按钮在 gameserver 镜像里已预装，通常不需要再点。游戏服的安装/更新通过面板的「安装/更新游戏服」完成。

## 环境变量

### paladmin 容器
| 变量 | 默认值 | 说明 |
|---|---|---|
| `GAMESERVER_CONTAINER` | `palworld-gameserver` | 要管控的游戏服容器名 |
| `GAMESERVER_URL` | `http://gameserver:8212` | 游戏服 REST API 地址 |
| `PALADIN_DATA_DIR` | `/data` | 面板数据目录 |

### gameserver 容器
| 变量 | 默认值 | 说明 |
|---|---|---|
| `STEAMAPPID` | `2394010` | 帕鲁服务端 App ID |
| `PROTON_VERSION` | `GE-Proton9-21` | GE-Proton 版本 |

## 常用命令

```bash
# 启动/停止/重启
docker compose up -d
docker compose down
docker compose restart paladmin

# 查看游戏服日志（也可在面板看）
docker compose logs -f gameserver

# 手动更新游戏服
docker compose exec gameserver /home/steam/entrypoint.sh update

# 进入游戏服容器排查
docker compose exec gameserver bash

# 备份游戏存档（宿主机执行）
tar -czf backup-$(date +%F).tar.gz ./data/gameserver/SaveGames
```

## 数据持久化

| 宿主机路径 | 容器路径 | 内容 |
|---|---|---|
| `./data/paladmin` | `/data` | 面板数据库 pst.db、配置 |
| `./data/gameserver` | `/home/steam/palserver` | 游戏服安装目录、存档、配置、PalDefender |

更新镜像不会丢失数据（卷保留）。建议定期备份 `./data` 目录。

## PalDefender 反作弊

gameserver 镜像已预置 GE-Proton。PalDefender DLL 需要在面板「PalDefender」页点击安装，会下载到：
```
./data/gameserver/Pal/Binaries/Win64/{d3d9.dll,PalDefender.dll}
```
启动游戏服时通过 `WINEDLLOVERRIDES=d3d9=n,b` 自动注入。

反作弊开关（踢出/封禁/IP封禁）在「设置」页配置，保存时写入 PalDefender/Config.json 并热重载。

## 故障排查

**游戏服启动失败**：
```bash
docker compose logs gameserver
```
常见原因：内存不足（Proton 需要较多内存）、端口被占用、文件权限问题。

**面板连不上游戏服**：
- 确认 gameserver 容器在运行：`docker compose ps`
- 确认游戏配置开启了 REST API 且端口 8212
- 在 paladmin 容器内测试连通性：`docker compose exec paladmin curl http://gameserver:8212/v1/api/version`

**面板无法启停游戏服**：
- 确认 `/var/run/docker.sock` 已挂载（默认已挂）
- 确认 paladmin 容器内有 docker 命令（镜像已装 docker.io）
- 容器名与 `GAMESERVER_CONTAINER` 一致

**Proton 相关错误**：
gameserver 镜像基于 Debian bookworm，已装 32/64 位库。若仍报缺少库，进入容器安装：
```bash
docker compose exec gameserver dpkg --add-architecture i386
docker compose exec gameserver apt-get update
docker compose exec gameserver apt-get install -y <缺失的包>
```
