# PalAdmin

幻兽帕鲁（Palworld）服务器管理与反作弊面板。

- 🖥️ **服务器管理**：在线玩家、公会、背包/帕鲁数据、踢封禁、RCON 控制台、白名单、广播、关服
- 🎮 **SteamCMD 管理**：在面板填写 SteamCMD 路径，一键安装/更新服务端，启动、停止、重启，无需手写命令
- 🛡️ **反作弊**：帕鲁属性/天赋/灵魂越界、Boss 帕鲁、复制帕鲁、非法物品、堆叠异常、瞬移、同 IP 多开等检测，支持警告/踢出/封禁/IP 封禁
- 💾 **存档备份与回档**：定时备份，一键回滚到任意备份（自动停服、安全备份、恢复、启服）
- ⚙️ **可视化配置**：游戏连接、密码、反作弊开关、Webhook 通知等全部在面板设置，即时生效
- 🔔 **告警通知**：钉钉 / 企业微信 / Discord / 通用 Webhook，含证据与审计日志
- 🐧 **纯 Linux 原生**：无需 Wine 或 Windows 兼容层

## 面板架构

```
┌──────────────────────────────────────────────────────┐
│                    浏览器（Vue3 面板）                │
└───────────────────────┬──────────────────────────────┘
                        │ http://服务器IP:8190
┌───────────────────────▼──────────────────────────────┐
│                  PalAdmin（宿主机进程）               │
│  管理 API · 定时任务 · 反作弊引擎 · 存档解析          │
│         └── 调用 SteamCMD 安装/更新游戏服 ──┐        │
└───────────────────────┬─────────────────────│────────┘
                        │                     │
┌───────────────────────▼──────────────┐  ┌───▼──────────────┐
│  PalServer 游戏进程（面板启停）       │  │ 存档/备份/证据    │
│  REST :8212   RCON :25575           │  │ /var/lib/paladmin│
└──────────────────────────────────────┘  └──────────────────┘
```

- PalAdmin 以二进制方式运行在宿主机（也支持 Docker），直接调用 SteamCMD 安装/更新游戏服
- 游戏服进程由面板启停，数据保存在 `/var/lib/paladmin/`
- 面板密码、游戏连接、反作弊规则等全部保存在数据库，面板网页可改

## 部署

PalAdmin 以二进制方式直接安装到宿主机（非 Docker），由 systemd 托管，可直接调用宿主机上的 SteamCMD 与游戏进程。

### 前置条件

- Linux 服务器（推荐 Ubuntu 22.04 / Debian 12）
- 服务器开放端口：面板 `8190/tcp`、游戏 `8211/udp`、REST `8212/tcp`、RCON `25575/tcp`（后两者建议仅内网）

### 第一步：安装 SteamCMD

Ubuntu / Debian：

```bash
sudo add-apt-repository multiverse -y
sudo dpkg --add-architecture i386
sudo apt update
sudo apt install -y steamcmd lib32gcc-s1
mkdir -p ~/steamcmd ~/PalServer
```

CentOS / RHEL 等请参考 SteamCMD 官方文档。Windows 下载 SteamCMD 压缩包解压即可。

### 第二步：安装 PalAdmin

提供两种方式，**推荐方式一（二进制直装）**，可直接调用宿主机 SteamCMD。

#### 方式一：二进制 + systemd（推荐）

一键脚本（自动从 GitHub Release 下载对应架构二进制、创建用户、注册 systemd 服务）：

```bash
curl -fsSL https://raw.githubusercontent.com/QYC-qyc/palworld-tool/main/scripts/install.sh | sudo bash
```

脚本会：
- 依次尝试多个公共 GitHub 镜像，自动选择可用的下载源（支持 aria2 多线程）
- 从 GitHub Release 下载对应架构的二进制
- 安装到 `/opt/paladmin/`，目录结构：
  ```
  /opt/paladmin/
  ├── paladmin          # 主程序
  ├── sav_cli           # 存档解析工具
  ├── web/              # 前端资源（index.html、assets/）
  ├── data/gamedata/    # 游戏数据（pal/item/passive ids 等）
  ├── paladmin.service
  └── config.yaml
  ```
- 数据（数据库、备份、证据、日志）存放在 `/var/lib/paladmin/`
- 注册并启动 `paladmin` 系统服务
- 自动选择 amd64 / arm64 架构

> 若所有公共镜像均不可用，可配置 HTTP 代理（见下方"下载加速"），或手动下载 tar 包后传到服务器。
> 二进制由 GitHub Actions 自动构建。每个正式版本（tag，如 `v0.2.4`）都会附带 `paladmin_linux_amd64.tar.gz` 与 `paladmin_linux_arm64.tar.gz`。

**启动验证：**

```bash
sudo systemctl status paladmin      # 状态应为 active (running)
journalctl -u paladmin -f           # 查看日志，应出现：
                                    #   前端目录: web
                                    #   游戏数据加载完成: N 帕鲁, M 物品...
                                    #   监听: http://0.0.0.0:8190
```

访问 `http://服务器IP:8190`，应显示登录页。

**更新到新版本：**

```bash
# 重新运行安装脚本即可（配置和数据保留）
curl -fsSL https://raw.githubusercontent.com/QYC-qyc/palworld-tool/main/scripts/install.sh | sudo bash
sudo systemctl restart paladmin
```

#### 方式二：Docker 部署

如果偏好容器化，用 Docker 运行面板（需把宿主机 SteamCMD 与游戏目录挂载进容器）：

```bash
sudo mkdir -p /www/palworld-tool && cd /www/palworld-tool
sudo curl -o docker-compose.yml \
  https://raw.githubusercontent.com/QYC-qyc/palworld-tool/main/docker-compose.yml
sudo docker compose up -d
```

> Docker 方式下，面板通过挂载的 `/opt/steamcmd`、`/opt/palserver` 调用 SteamCMD；
> 若你的路径不同，编辑 `docker-compose.yml` 修改挂载。游戏服进程在容器内运行，使用 host 网络。

### 第三步：初始化与配置游戏服

1. 浏览器访问 `http://你的服务器IP:8190`，首次进入设置**面板登录密码**
2. 进入左侧「**游戏服**」菜单，填写：
   - **SteamCMD 路径**：如 `/usr/games/steamcmd`（Ubuntu）或 `/home/steam/steamcmd/steamcmd.sh`
   - **游戏安装目录**：如 `/home/steam/PalServer`
   - 可选：启动额外参数、游戏端口
   - 点击「保存配置」
3. 点击「**安装 / 更新游戏服**」
   - 面板执行 `steamcmd +force_install_dir <目录> +login anonymous +app_update 2394010 validate +quit`
   - 实时进度显示在日志区，首次约几分钟
4. 安装完成后点击「**启动**」运行游戏服
5. 进入「**游戏配置**」（`.ini`），确认游戏服开启了 REST API：
   - `RESTAPIEnabled=true`、`RESTAPIPort=8212`、`AdminPassword=...`
   - 配置保存后重启游戏服生效
6. 进入「**系统设置**」，填写游戏服 REST 地址（如 `http://127.0.0.1:8212`）和 AdminPassword，面板即可同步在线玩家、执行 RCON 等

> 首次启动时日志可能出现"备份失败"、"同步在线玩家失败"，这是因为游戏服尚未安装/配置，完成上述步骤后会消失。

### 防火墙

```bash
sudo ufw allow 8190/tcp
sudo ufw allow 8211/udp
```

云服务器还需在安全组放行相同端口。

### 常用运维命令

**二进制方式：**

```bash
sudo systemctl status paladmin      # 状态
sudo systemctl restart paladmin     # 重启
journalctl -u paladmin -f           # 日志
# 更新 PalAdmin：重新运行安装脚本
curl -fsSL https://raw.githubusercontent.com/QYC-qyc/palworld-tool/main/scripts/install.sh | sudo bash
```

**Docker 方式：**

```bash
cd /www/palworld-tool
docker compose logs -f paladmin     # 日志
docker compose restart              # 重启
docker compose pull && docker compose up -d   # 更新
```

游戏服本身的更新统一在面板「游戏服」页点击「安装/更新」。

## 使用说明

| 功能 | 说明 |
|---|---|
| 仪表盘 | 在线人数、服务器 FPS、快捷操作、反作弊统计 |
| 游戏服 | 配置 SteamCMD 路径，安装/更新服务端，启动/停止/重启，查看实时日志 |
| 玩家 | 在线/存档玩家列表，查看背包、帕鲁，执行踢封禁 |
| 公会 | 公会信息与成员 |
| 封禁列表 | 管理玩家与 IP 封禁 |
| RCON 控制台 | 在线执行游戏服命令 |
| 备份管理 | 存档备份列表，支持一键回档 |
| 反作弊告警 | 查看检测记录、证据，人工标记处理 |
| 反作弊规则 | 开关各检测项、调整处置动作 |
| 系统设置 | 游戏连接、面板密码、Webhook 等配置 |

> **首次使用建议**：先在「反作弊规则」中关闭自动封禁/踢出，只保留警告，观察一段时间确认检测准确后再开启处罚。

## 致谢

- [palworld-server-tool](https://github.com/zaigie/palworld-server-tool) —— 存档解析与管理框架参考
- [PalDefender](https://github.com/Ultimeit/PalDefender) —— 反作弊设计理念参考
- [palworld-save-tools](https://github.com/cheahjs/palworld-save-tools) —— 存档格式解析
- 帕鲁 ID 数据参考 [paldeck.cc](https://paldeck.cc)

## 许可证

本项目采用 MIT 许可证。
