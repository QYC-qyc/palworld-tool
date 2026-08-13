# PalAdmin

幻兽帕鲁（Palworld）服务器管理与反作弊面板。

- 🖥️ **服务器管理**：在线玩家、公会、背包/帕鲁数据、踢封禁、RCON 控制台、白名单、广播、关服
- 🎮 **游戏服一键部署**：面板内安装、启动、停止、重启、更新幻兽帕鲁服务端，无需手写命令
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
│                    PalAdmin 容器                      │
│  管理 API  ·  定时任务  ·  反作弊引擎  ·  存档解析     │
│         └── 通过 /var/run/docker.sock 管理游戏服 ──┐  │
└───────────────────────┬────────────────────────────│──┘
                        │                            │
┌───────────────────────▼──────────────┐  ┌─────────▼─────────┐
│     palworld 游戏服容器（面板创建）    │  │  存档/备份/证据卷  │
│     REST :8212   RCON :25575         │  │  /www/palworld-tool│
└───────────────────────────────────────┘  └────────────────────┘
```

- PalAdmin 以镜像方式运行，游戏服由面板通过 Docker 一键部署与管理
- 面板密码、游戏连接、反作弊规则等全部保存在数据库，面板网页可改
- 数据统一存放在宿主机 `/www/palworld-tool/` 目录

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

一键脚本（自动下载对应架构二进制、创建用户、注册 systemd 服务）：

```bash
curl -fsSL https://gitee.com/QYC-qyc/palworld-tool/raw/main/scripts/install.sh | sudo bash
```

脚本会：
- 安装到 `/opt/paladmin/`（含 paladmin、sav_cli、前端资源）
- 数据存放在 `/var/lib/paladmin/`
- 注册并启动 `paladmin` 系统服务
- 自动选择 amd64 / arm64 架构

### 第三步：初始化与配置游戏服

1. 浏览器访问 `http://你的服务器IP:8190`，首次进入设置**面板登录密码**
2. 进入左侧「**游戏服**」菜单，填写：
   - **SteamCMD 路径**：如 `/usr/games/steamcmd`（Ubuntu）或 `/home/steam/steamcmd/steamcmd.sh`
   - **游戏安装目录**：如 `/home/steam/PalServer`
   - 可选：启动额外参数
3. 点击「**安装 / 更新游戏服**」
   - 面板执行 `steamcmd +login anonymous +app_update 2394010 validate +quit`
   - 实时进度显示在日志区，首次约几分钟
4. 安装完成后点击「**启动**」运行游戏服

### 防火墙

```bash
sudo ufw allow 8190/tcp
sudo ufw allow 8211/udp
```

云服务器还需在安全组放行相同端口。

### 常用运维命令

```bash
# 服务管理
sudo systemctl status paladmin
sudo systemctl restart paladmin
sudo systemctl stop paladmin

# 查看日志
journalctl -u paladmin -f

# 更新 PalAdmin（重新运行安装脚本即可）
curl -fsSL https://gitee.com/QYC-qyc/palworld-tool/raw/main/scripts/install.sh | sudo bash

# 更新游戏服：在面板「游戏服」页点击「安装/更新」
```

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
