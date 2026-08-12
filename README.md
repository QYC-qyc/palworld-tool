# PalAdmin

幻兽帕鲁（Palworld）服务器管理与反作弊系统，参考 [palworld-server-tool](https://github.com/zaigie/palworld-server-tool) 设计。纯 Linux 原生方案，无需 Wine。

- 🖥️ **服务器管理**：在线玩家监控、玩家/公会/背包/帕鲁数据、踢封禁、RCON 控制台、白名单、广播、关服
- 🛡️ **反作弊引擎**：12 条存档规则 + 5 条在线规则，四级处置阶梯（警告 → 踢出 → 封禁 → IP 封禁）
- 🔍 **检测能力**：帕鲁属性/天赋/灵魂越界、Boss/塔主帕鲁、复制帕鲁（InstanceID + 特征指纹）、非法物品、物品堆叠异常、瞬移、同 IP 多开、等级突变
- 🐧 **纯 Linux 原生**：无需 Wine 或 Windows 兼容层，原生 Linux 游戏服直接运行
- 💾 **存档回档**：停服 → 自动安全备份 → 恢复 → 启服，支持 docker 进程控制
- ⚙️ **面板动态设置**：连接信息、密码、反作弊开关等可在网页修改即时生效，无需改文件重启
- 🔔 **告警通知**：Webhook 推送（钉钉 / 企业微信 / Discord / 通用），证据留存与审计日志
- 🐳 **轻量部署**：单容器镜像（含前端、后端、存档解析），推送到 GHCR，服务器 `docker pull` 即用；bbolt 单文件数据库，无外部依赖

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.22, Gin, bbolt, viper, zap, gocron |
| 前端 | Vue 3, Vite, TypeScript, Naive UI, Pinia |
| 存档解析 | Python + palworld-save-tools（sav_cli） |

## 部署流程

目标系统：**Ubuntu 22.04**（其他 Linux 同理）。采用**镜像部署**：推代码后 GitHub Actions 自动构建镜像，服务器直接 `docker pull` 运行，无需在服务器安装 Go/Node/Python。

镜像地址（**公开，免登录拉取**）：

```
ghcr.io/qyc-qyc/palworld-tool:latest
```

代码仓库：<https://github.com/QYC-qyc/palworld-tool>

> 反作弊为**纯 Linux 原生方案**，不依赖 Wine 或 PalDefender：通过定期解析存档 + 官方 REST/RCON 在线监控实现检测与处置，原生 Linux 游戏服直接可用。
>
> 游戏服需在 `PalWorldSettings.ini` 开启：`RESTAPIEnabled=True`、`RESTAPIPort=8212`、`RCONEnabled=True`、`RCONPort=25575`，并设置 `AdminPassword`。

---

### 一、镜像自动构建（推代码即可）

项目配置了 **GitHub Actions**：`git push` 到 `main` 后自动在云端构建镜像并推送到 GHCR，无需本地 Docker：

```bash
git add .
git commit -m "更新"
git push origin main
```

- 工作流：`.github/workflows/docker.yml`
- 进度查看：GitHub 仓库 → **Actions**
- 约 2-5 分钟后镜像更新到 `ghcr.io/qyc-qyc/palworld-tool`
- 镜像标签：`latest`（main）、版本号、提交短 SHA

> 也可用 `scripts/docker-push.sh/.ps1` 在本地手动构建推送。

---

### 二、服务器拉取并启动

`docker-compose.yml` 同时包含 `palworld`（原生 Linux 游戏服）和 `paladmin`（面板），所有数据统一放在 `/www/palworld-tool/`。

```bash
# 1. 创建目录
sudo mkdir -p /www/palworld-tool && cd /www/palworld-tool

# 2. 下载 compose
sudo curl -o docker-compose.yml \
  https://raw.githubusercontent.com/QYC-qyc/palworld-tool/main/docker-compose.yml

# 3. 下载示例配置（作为容器内默认配置，密码用 .env 覆盖）
sudo curl -o config.yaml \
  https://raw.githubusercontent.com/QYC-qyc/palworld-tool/main/config.example.yaml

# 4. 配置密码
cat > .env <<'EOF'
WEB_PASSWORD=你的面板强密码
GAME_ADMIN_PASSWORD=你想设置的游戏AdminPassword
EOF

# 5. 启动（首次会拉取游戏服镜像并自动安装游戏）
sudo docker compose up -d
sudo docker compose logs -f
```

数据目录结构：

```
/www/palworld-tool/
├── docker-compose.yml
├── .env / config.yaml
├── pst.db             # 数据库（自动生成）
├── data/ backups/ evidence/ logs/   # 面板数据
└── game/              # 游戏服安装目录与存档（容器自动写入）
```

compose 关键配置：

```yaml
services:
  palworld:
    image: thijsvanloef/palworld-server-docker:latest
    volumes:
      - /www/palworld-tool/game:/palworld
  paladmin:
    image: ghcr.io/qyc-qyc/palworld-tool:latest
    ports:
      - "8190:8190"
    volumes:
      - /www/palworld-tool/game/Pal/Saved:/game/Saved
    environment:
      WEB__PASSWORD: "${WEB_PASSWORD}"
      REST__ADDRESS: "http://palworld:8212"
      REST__PASSWORD: "${GAME_ADMIN_PASSWORD}"
      RCON__ADDRESS: "palworld:25575"
      RCON__PASSWORD: "${GAME_ADMIN_PASSWORD}"
      SAVE__PATH: "/game/Saved/SaveGames/0"
      PROCESS__MODE: "docker"
      PROCESS__CONTAINER: "palworld"
```

启动后浏览器访问 `http://服务器IP:8190`，**首次进入会自动跳转到初始化向导**，设置面板登录密码（可同时填写游戏 REST/RCON 连接信息，也可稍后在「系统设置」配置）。完成后用该密码登录。

> 不再需要手动挂载 config.yaml 或预先设置密码；所有配置保存在数据库中，面板可改。
> 若游戏服已单独部署，在向导或系统设置里把 REST/RCON 地址改成实际地址（如 `http://172.17.0.1:8212`）。

---

### 防火墙

```bash
sudo ufw allow 8190/tcp     # 面板端口
sudo ufw allow 8211/udp     # 游戏端口
# 游戏服的 8212(REST)、25575(RCON) 不要对公网开放
```

> 公网使用强烈建议前面加 Nginx/Caddy 反向代理并启用 HTTPS；REST/RCON 仅绑定内网或经反代鉴权。

---

### 部署后检查清单

1. 登录面板 →「系统设置」填好连接信息并保存
2. 仪表盘显示在线人数 / FPS = REST 对接成功
3. 等待约 2 分钟，玩家页出现存档数据 = 存档解析正常
4. **首次先关闭自动封禁**验证检测：「系统设置」把 kick/ban 关闭，只留 warn
5. 按 [docs/联调清单.md](docs/联调清单.md) 完成端到端验证
6. 确认检测无误后再开启 kick/ban

### 更新版本

代码 `git push` 后，GitHub Actions 会自动构建并推送新镜像（约 2-5 分钟）。完成后在服务器拉取并重启：

```bash
docker compose pull
docker compose up -d
```

可以在 GitHub 仓库 → Actions 页确认构建状态。

## 配置

主要配置（均也可在面板「系统设置」动态修改）：

```yaml
web:
  password: "CHANGE_ME"     # 面板密码 / JWT 密钥
  port: 8190
rest:
  address: "http://127.0.0.1:8212"   # 游戏服 REST API
  password: "游戏AdminPassword"
rcon:
  address: "127.0.0.1:25575"
save:
  path: "/path/to/Saved/SaveGames/0/<GUID>"   # Level.sav 所在目录
anticheat:
  enabled: true
  mode: "external"          # 纯 Linux 原生反作弊
  punish:
    warn: true
    kick: false
    ban: true               # 首次建议先关闭，验证检测准确性
process:
  mode: "docker"            # noop / docker（回档需要，控制游戏容器停启）
  container: "palworld"
```

## 截图

|  |  |
|---|---|
| 仪表盘：在线人数、FPS、快捷操作、反作弊统计 | 玩家：在线/存档列表、详情、踢封禁 |
| 反作弊告警：证据查看与人工处置 | 系统设置：连接与反作弊动态配置 |

## 项目结构

```
.
├── api/            # Gin HTTP handlers
├── internal/
│   ├── config/     # 配置加载与动态应用
│   ├── database/   # bbolt 初始化与数据模型
│   ├── tool/       # 官方 REST / RCON / Webhook 客户端
│   ├── task/       # 定时任务（在线/存档/备份同步）
│   ├── executor/   # RCON 执行器
│   ├── source/     # 存档来源（本地，预留 http/docker）
│   ├── system/     # 网络/文件/进程控制
│   └── logger/     # zap 日志
├── service/        # 业务逻辑层
│   └── anticheat/  # 反作弊引擎：规则、扫描器、处置、gamedata
├── module/         # Python sav_cli 存档解析器
├── web/            # Vue3 前端
├── data/gamedata/  # 合法数据表（帕鲁/物品/词条/限制）
├── scripts/        # 镜像推送脚本（docker-push）
├── Dockerfile      # 多阶段镜像构建
├── docker-compose.yml
└── docs/           # 开发文档与联调清单
```

## 反作弊规则

### 存档类（S 系列）

| ID | 检测项 |
|---|---|
| S001 | 帕鲁属性越界（等级/阶级/HP/攻击/防御/类型） |
| S002 | 非法天赋(IV)与帕鲁灵魂强化 |
| S003 | 非法被动词条（数量/空值/合法性） |
| S004 | 非法持有 Boss/塔主帕鲁 |
| S005 | 物品堆叠越界 |
| S006 | 非法/调试物品 |
| S007 | 复制帕鲁（InstanceID 精确匹配 + 特征指纹） |
| S008 | 玩家属性越界 |
| S009 | 经验异常 |
| S010 | 进度异常增长 |
| S011 | 非法据点 |
| S012 | 资源/金钱异常 |

### 在线类（L 系列）

| ID | 检测项 |
|---|---|
| L001 | 移动速度异常 / 瞬移 |
| L002 | 同 IP 多账号在线 |
| L003 | 账号重复登录 |
| L004 | 在线等级突变 |
| L005 | 非白名单玩家在线 |

规则可在面板「反作弊规则」页开关，处置动作可配置。

## 开发

```bash
# 后端
go run . --config config.yaml

# 前端（热更新，自动代理 /api 到 8080）
cd web && npm install && npm run dev

# 测试
go test ./...

# 本地构建镜像
docker build -t paladmin:local .
```

推送镜像到 GHCR：`bash scripts/docker-push.sh latest`（Windows 用 `docker-push.ps1`）。

## 致谢

- [palworld-server-tool](https://github.com/zaigie/palworld-server-tool) —— 存档解析与基础管理框架参考
- [palworld-save-tools](https://github.com/cheahjs/palworld-save-tools) —— 存档格式解析
- ID 数据参考 [paldeck.cc](https://paldeck.cc)

## 许可证

本项目采用 MIT 许可证。
