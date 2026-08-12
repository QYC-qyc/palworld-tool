# PalAdmin

幻兽帕鲁（Palworld）服务器管理与反作弊系统，参考 [palworld-server-tool](https://github.com/zaigie/palworld-server-tool) 与 [PalDefender](https://github.com/Ultimeit/PalDefender) 设计。

- 🖥️ **服务器管理**：在线玩家监控、玩家/公会/背包/帕鲁数据、踢封禁、RCON 控制台、白名单、广播、关服
- 🛡️ **反作弊引擎**：12 条存档规则 + 5 条在线规则，四级处置阶梯（警告 → 踢出 → 封禁 → IP 封禁）
- 🔍 **检测能力**：帕鲁属性/天赋/灵魂越界、Boss/塔主帕鲁、复制帕鲁（InstanceID + 特征指纹）、非法物品、物品堆叠异常、瞬移、同 IP 多开、等级突变
- 🔌 **PalDefender 集成**：可选对接其进程内实时反作弊 REST API（`:17993`），外置/集成双模式
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

目标系统：**Ubuntu 22.04**（其他 Linux 同理）。仅支持**镜像部署**：在开发机构建镜像推送到 GitHub Container Registry (GHCR)，服务器直接 `docker pull` 运行，无需在服务器安装 Go/Node/Python。

镜像地址：

```
ghcr.io/qyc-qyc/palworld-tool:latest
```

代码仓库：<https://github.com/QYC-qyc/palworld-tool>

> 游戏服需在 `PalWorldSettings.ini` 开启：`RESTAPIEnabled=True`、`RESTAPIPort=8212`、`RCONEnabled=True`、`RCONPort=25575`，并设置 `AdminPassword`。

---

### 一、构建并推送镜像（开发机，执行一次）

需要本机安装 Docker。

1. 在 GitHub 生成 Token：Settings → Developer settings → Personal access tokens → **Tokens (classic)** → Generate new token，勾选 **`write:packages`** 权限。

2. 构建并推送：

   ```bash
   # Linux / macOS
   export GITHUB_USER=QYC-qyc
   export GITHUB_TOKEN=你的GitHubToken
   bash scripts/docker-push.sh latest
   ```
   ```powershell
   # Windows PowerShell
   $env:GITHUB_TOKEN="你的GitHubToken"
   powershell -ExecutionPolicy Bypass -File scripts\docker-push.ps1
   ```

   镜像由 `Dockerfile` 多阶段构建（前端→后端→sav_cli），推送到 GHCR。

3. （可选）把镜像改为公开：GitHub 个人主页 → Packages → `palworld-tool` → Package settings → Change visibility → Public。公开后服务器 `docker pull` 无需登录。

---

### 二、服务器拉取并启动

#### 方式 A：外置模式（原生 Linux 游戏服，最简单）

```bash
mkdir -p paladmin && cd paladmin

# 获取 compose 模板
curl -o docker-compose.yml \
  https://raw.githubusercontent.com/QYC-qyc/palworld-tool/main/docker-compose.yml

# 配置环境变量
cat > .env <<'EOF'
WEB_PASSWORD=你的面板强密码
GAME_ADMIN_PASSWORD=游戏AdminPassword
EOF

# 修改存档挂载路径
nano docker-compose.yml   # 把 /home/steam/Pal/Saved 改成真实路径
```

`docker-compose.yml` 关键配置：

```yaml
services:
  paladmin:
    image: ghcr.io/qyc-qyc/palworld-tool:latest
    ports:
      - "8190:8190"
    volumes:
      - /你的真实路径/Pal/Saved:/game/Saved   # 回档需要读写，勿加 :ro
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

启动：

```bash
# 私有镜像先登录：docker login ghcr.io -u QYC-qyc
docker compose up -d
docker compose logs -f paladmin
```

访问 `http://服务器IP:8190`，用 `WEB_PASSWORD` 登录。

> 如果游戏服不是容器而是装在主机上，把 `palworld` 换成宿主机地址（如 `http://172.17.0.1:8212`、`172.17.0.1:25575`），并把游戏加入同一网络或用 `network_mode: host`。

#### 方式 B：集成模式（Linux 游戏服 + PalDefender/Wine）

若游戏服通过 Wine 运行 PalDefender：

```bash
curl -o docker-compose.paldefender.yml \
  https://raw.githubusercontent.com/QYC-qyc/palworld-tool/main/docker-compose.paldefender.yml
# 准备 ./PalDefender（PalDefender.dll、d3d9.dll、RESTAPI/Tokens）
echo 'PALDEFENDER_TOKEN=你的PDToken' >> .env
docker compose -f docker-compose.paldefender.yml up -d
```

该 compose 包含：
- `palworld`：原生 Linux 游戏服
- `paldefender`：Wine 容器运行 PalDefender（仅它需要 Wine）
- `paladmin`：拉取 `ghcr.io/qyc-qyc/palworld-tool`，`ANTICHEAT__MODE=integrated`

面板「PalDefender」页显示绿色已连接即成功。

| 能力 | 外置 (A) | 集成 (B) |
|---|---|---|
| 存档/复制/非法物品检测 | ✅ | ✅ |
| 瞬移/多开检测 | ✅ | ✅ |
| 进程内实时伤害/非法 stat 拦截 | ❌ | ✅ |
| PalDefender 私聊/IP 封禁 | ❌ | ✅ |
| 额外需要 Wine | ❌ | ✅（仅 PalDefender 容器） |

---

### 防火墙

```bash
sudo ufw allow 8190/tcp     # 面板端口
sudo ufw allow 8211/udp     # 游戏端口
# 游戏服的 8212(REST)、25575(RCON)、PalDefender 17993 不要对公网开放
```

> 公网使用强烈建议前面加 Nginx/Caddy 反向代理并启用 HTTPS；REST/RCON/17993 仅绑定内网或经反代鉴权。

---

### 部署后检查清单

1. 登录面板 →「系统设置」填好连接信息并保存
2. 仪表盘显示在线人数 / FPS = REST 对接成功
3. 等待约 2 分钟，玩家页出现存档数据 = 存档解析正常
4. **首次先关闭自动封禁**验证检测：「系统设置」把 kick/ban 关闭，只留 warn
5. 按 [docs/联调清单.md](docs/联调清单.md) 完成端到端验证
6. 确认检测无误后再开启 kick/ban

### 更新版本

在开发机推送新镜像后，服务器拉取最新镜像并重启：

```bash
docker compose pull
docker compose up -d

# PalDefender 集成模式：
docker compose -f docker-compose.paldefender.yml pull
docker compose -f docker-compose.paldefender.yml up -d
```

开发机推送：`bash scripts/docker-push.sh latest`（或 Windows 的 `docker-push.ps1`）。

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
  mode: "external"          # external（存档扫描）/ integrated（对接 PalDefender）
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
│   ├── tool/       # 官方 REST / RCON / PalDefender / Webhook 客户端
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
├── docker-compose.yml         # 外置模式（拉镜像）
├── docker-compose.paldefender.yml  # PalDefender 集成模式
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
- [PalDefender](https://github.com/Ultimeit/PalDefender) —— 反作弊设计理念、处置阶梯、REST API 参考
- [palworld-save-tools](https://github.com/cheahjs/palworld-save-tools) —— 存档格式解析
- ID 数据参考 [paldeck.cc](https://paldeck.cc)

## 许可证

本项目采用 MIT 许可证。
