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

### 前置条件

- Linux 服务器（推荐 Ubuntu 22.04）
- 已安装 Docker 与 Docker Compose
- 服务器开放端口：面板 `8190/tcp`、游戏 `8211/udp`

### 第一步：准备目录并启动面板

```bash
sudo mkdir -p /www/palworld-tool && cd /www/palworld-tool
```

下载 `docker-compose.yml`（两种方式任选其一）：

**方式一：从 Gitee 下载**

```bash
sudo curl -o docker-compose.yml \
  https://gitee.com/QYC-qyc/palworld-tool/raw/main/docker-compose.yml
```

**方式二：自行创建**

```bash
sudo nano docker-compose.yml
```

填入以下内容并保存：

```yaml
services:
  paladmin:
    image: gitee.com/qyc-qyc/paladmin:latest
    container_name: paladmin
    restart: unless-stopped
    ports:
      - "8190:8190"
    volumes:
      - pst-data:/app/data
      - /www/palworld-tool/backups:/app/backups
      - /www/palworld-tool/evidence:/app/evidence
      - /www/palworld-tool/logs:/app/logs
      - /var/run/docker.sock:/var/run/docker.sock
      - /www/palworld-tool/game:/game/Saved
    environment:
      TZ: "Asia/Shanghai"
      STORAGE__PATH: "/app/data/pst.db"
      SAVE__DECODE_PATH: "/app/sav_cli"
      SAVE__PATH: "/game/Saved/SaveGames/0"
      ANTICHEAT__ENABLED: "true"
      ANTICHEAT__MODE: "external"
      PROCESS__MODE: "docker"
      PROCESS__CONTAINER: "palworld"
    networks:
      - palnet

volumes:
  pst-data:

networks:
  palnet:
    driver: bridge
```

启动面板：

```bash
sudo docker compose up -d
sudo docker compose logs -f paladmin
```

### 第二步：初始化面板

浏览器访问 `http://你的服务器IP:8190`，首次进入会自动跳转到初始化向导：

1. 设置**面板登录密码**
2. 游戏连接信息可稍后在「系统设置」配置

### 第三步：部署游戏服

1. 进入左侧「**游戏服**」菜单
2. 填写游戏管理员密码（AdminPassword）、服务器名称、端口
3. 点击「**部署并启动**」
   - 面板自动拉取幻兽帕鲁服务端镜像、创建容器、启动服务
   - 首次启动需要下载服务端文件，请等待几分钟
4. 部署完成后可在同一页面**启动 / 停止 / 重启 / 更新**游戏服，并查看实时日志

游戏数据保存在 `/www/palworld-tool/game`，更新或重建容器不会丢失存档。

### 防火墙

```bash
sudo ufw allow 8190/tcp
sudo ufw allow 8211/udp
```

云服务器还需在安全组放行相同端口。游戏服的 REST（8212）和 RCON（25575）仅供面板内部使用，**不要对公网开放**。

### 更新面板

```bash
cd /www/palworld-tool
sudo docker compose pull
sudo docker compose up -d
```

游戏服更新在面板「游戏服」页面点击「更新」即可。

## 使用说明

| 功能 | 说明 |
|---|---|
| 仪表盘 | 在线人数、服务器 FPS、快捷操作、反作弊统计 |
| 游戏服 | 一键部署/启动/停止/重启/更新游戏服，查看日志 |
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
