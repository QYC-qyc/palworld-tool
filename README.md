# PalAdmin

幻兽帕鲁（Palworld）服务器管理与反作弊系统，参考 [palworld-server-tool](https://github.com/zaigie/palworld-server-tool) 与 [PalDefender](https://github.com/Ultimeit/PalDefender) 设计。

- 🖥️ **服务器管理**：在线玩家监控、玩家/公会/背包/帕鲁数据、踢封禁、RCON 控制台、白名单、广播、关服
- 🛡️ **反作弊引擎**：12 条存档规则 + 5 条在线规则，四级处置阶梯（警告 → 踢出 → 封禁 → IP 封禁）
- 🔍 **检测能力**：帕鲁属性/天赋/灵魂越界、Boss/塔主帕鲁、复制帕鲁（InstanceID + 特征指纹）、非法物品、物品堆叠异常、瞬移、同 IP 多开、等级突变
- 🔌 **PalDefender 集成**：可选对接其进程内实时反作弊 REST API（`:17993`），外置/集成双模式
- 💾 **存档回档**：停服 → 自动安全备份 → 恢复 → 启服，支持 systemd / docker 进程控制
- ⚙️ **面板动态设置**：连接信息、密码、反作弊开关等可在网页修改即时生效，无需改文件重启
- 🔔 **告警通知**：Webhook 推送（钉钉 / 企业微信 / Discord / 通用），证据留存与审计日志
- 🗄️ **轻量部署**：Go 纯静态二进制 + bbolt 单文件数据库，无外部依赖；前端 Vue3 + Naive UI 内嵌

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.22, Gin, bbolt, viper, zap, gocron |
| 前端 | Vue 3, Vite, TypeScript, Naive UI, Pinia |
| 存档解析 | Python + palworld-save-tools（sav_cli） |

## 部署流程

目标系统：**Ubuntu 22.04**（其他 Linux 发行版同理）。两种方式任选其一：

- **方式 A：二进制 + systemd**——最简单，适合游戏服直接装在主机上
- **方式 B：Docker**——适合游戏服也容器化运行

> 游戏服需在 `PalWorldSettings.ini` 开启：`RESTAPIEnabled=True`、`RESTAPIPort=8212`、`RCONEnabled=True`、`RCONPort=25575`，并设置 `AdminPassword`。

---

### 方式 A：二进制 + systemd

#### 1. 获取发布包

**从源码构建**（构建机需 Go 1.22+、Node 18+、Python 3）：

```bash
git clone https://gitee.com/QYC-qyc/palworld-tool.git paladmin
cd paladmin
bash scripts/build.sh          # 产出 dist/paladmin-<ver>-linux-amd64.tar.gz
```

**或直接下载** [Releases](../../releases) 中的 `paladmin-linux-amd64.tar.gz`。

#### 2. 解压到服务器

```bash
sudo mkdir -p /opt && sudo tar -xzf paladmin-*-linux-amd64.tar.gz -C /opt
cd /opt/paladmin
```

#### 3. 安装运行依赖（存档解析需要 Python）

```bash
sudo apt update
sudo apt install -y python3 python3-pip
sudo pip3 install -r module/requirements.txt --break-system-packages \
  || sudo pip3 install -r module/requirements.txt
```

#### 4. 修改配置

```bash
cp config.example.yaml config.yaml
nano config.yaml
```

至少确认这几项（其余可登录后在面板「系统设置」里改）：

```yaml
web:
  password: "你的强密码"     # 面板登录密码
  port: 8190
rest:
  address: "http://127.0.0.1:8212"
  password: "游戏AdminPassword"
rcon:
  address: "127.0.0.1:25575"
  password: "游戏AdminPassword"
save:
  # 用 find / -name Level.sav 2>/dev/null 查找，取所在目录
  path: "/home/steam/Pal/Saved/SaveGames/0/<世界GUID>"
process:
  mode: "noop"             # 确认跑通后可改 systemd（回档需要）
```

#### 5. 注册并启动服务

```bash
sudo bash install.sh       # 自动创建 paladmin 用户、注册 systemd、开机自启
sudo systemctl status paladmin
sudo journalctl -u paladmin -f
```

#### 6. 验证与访问

```bash
curl http://127.0.0.1:8190/health    # 期望 {"status":"ok"}
```

浏览器打开 `http://服务器IP:8190` 登录。

---

### 方式 B：Docker

前提：服务器已安装 Docker，游戏服可通过容器名 `palworld` 访问。

```bash
git clone https://gitee.com/QYC-qyc/palworld-tool.git paladmin
cd paladmin

cp .env.example .env
nano .env                 # 设置 WEB_PASSWORD、GAME_ADMIN_PASSWORD
nano docker-compose.yml   # 把存档挂载路径改成真实路径
```

`docker-compose.yml` 关键配置：

```yaml
volumes:
  - /你的真实路径/Pal/Saved:/game/Saved   # 去掉 :ro 才能回档写入
environment:
  WEB__PASSWORD: "${WEB_PASSWORD}"
  REST__ADDRESS: "http://palworld:8212"
  REST__PASSWORD: "${GAME_ADMIN_PASSWORD}"
  RCON__ADDRESS: "palworld:25575"
  RCON__PASSWORD: "${GAME_ADMIN_PASSWORD}"
  SAVE__PATH: "/game/Saved"
  PROCESS__MODE: "docker"
  PROCESS__CONTAINER: "palworld"   # 游戏容器名，回档时控制停启
```

启动：

```bash
docker compose up -d --build
docker compose logs -f paladmin
```

访问 `http://服务器IP:8190`。

---

### 防火墙

```bash
sudo ufw allow 8190/tcp     # 面板端口
# 游戏服的 8212(REST)、25575(RCON) 不要对公网开放，保持 127.0.0.1
```

> 公网使用强烈建议前面加 Nginx/Caddy 反向代理并启用 HTTPS。

---

### 部署后检查清单

1. 登录面板 →「系统设置」填好连接信息并保存
2. 仪表盘显示在线人数 / FPS = REST 对接成功
3. 等待约 2 分钟，玩家页出现存档数据 = 存档解析正常
4. **首次先关闭自动封禁**验证检测准确性：
   `config.yaml` 中 `anticheat.punish.ban: false`、`kick: false`，只留 `warn`
5. 按 [docs/联调清单.md](docs/联调清单.md) 完成端到端验证
6. 确认检测无误后再开启 kick/ban

### 更新版本

```bash
# 二进制：重新解压发布包后
sudo systemctl restart paladmin

# Docker：
git pull && docker compose up -d --build
```

更多说明见 [DEPLOY.md](DEPLOY.md)。

---

### 方式 C：Docker 部署「Linux 游戏服 + PalDefender(Wine) + PalAdmin」

> **⚠️ 关于 PalDefender 的重要事实**
>
> [PalDefender](https://github.com/Ultimeit/PalDefender) 官方为 **Windows DLL**（通过 `d3d9.dll` 代理注入游戏进程），**不支持原生 Linux**。
>
> 但**只有 PalDefender 需要 Wine**——游戏服本身仍使用**官方原生 Linux 镜像**正常运行。
> 架构上是三个容器：
>
> | 服务 | 运行方式 | 作用 |
> |---|---|---|
> | `palworld` | **原生 Linux** 官方镜像 | 游戏服务器 |
> | `paldefender` | **Wine** 容器 | 运行 Windows 版 PalDefender，注入/防护 |
> | `paladmin` | Linux 容器 | 面板 + 反作弊，集成模式对接 PalDefender |
>
> - 若不使用 PalDefender → 游戏服原生 Linux 即可，PalAdmin 用**外置模式**。
> - 若使用 PalDefender → 额外用一个 Wine 容器运行它，游戏服仍是原生 Linux，PalAdmin 以**集成模式**对接。

下面使用 `docker-compose.paldefender.yml` 三服务编排。

#### 1. 准备 PalDefender 文件

从 [PalDefender Releases](https://github.com/Ultimeit/PalDefender/releases) 下载 `PalDefender_Windows.zip`，在项目目录解压为：

```
./PalDefender/
├── PalDefender.dll
├── d3d9.dll
├── RESTAPI/            # Token 配置（见下一步）
└── ...
```

> 这两个 DLL 会被挂载进游戏服的 `Pal/Binaries/Win64/` 目录，
> 由 `paldefender`（Wine）容器加载并注入游戏进程。

#### 2. 配置 PalDefender REST Token

在 `./PalDefender/RESTAPI/Tokens/` 目录建一个 token 文件（例如 `paladmin.json`）：

```json
{
  "Name": "PalAdmin",
  "Token": "生成一段随机长字符串",
  "Permissions": ["REST.*"]
}
```

该 Token 即 compose 中 `PALDEFENDER__TOKEN` 的值。最小权限可按需收窄为
`REST.Players.Read`、`REST.Punishments.*`、`REST.Messages.*` 等。
REST API 默认端口 `17993`（在 `paldefender` 服务内监听，不直接暴露公网）。

#### 3. 修改 compose 配置

```bash
cp docker-compose.paldefender.yml docker-compose.override.yml
nano docker-compose.override.yml
```

必须核对/修改：
- **palworld 服务** `image`：官方/社区**原生 Linux** 游戏服镜像（示例 `thijsvanloef/palworld-server-docker`，按你信任的镜像替换）
- **paldefender 服务** `image` / `command`：带 Wine 的镜像与启动命令（PalDefender 需以 Wine 加载 `PalServer-Win64-Shipping-Cmd.exe`，请按所选 Wine 镜像的实际路径调整）
- `ADMIN_PASSWORD` / `REST__PASSWORD` / `RCON__PASSWORD`：改为同一个强密码
- `WEB__PASSWORD`：PalAdmin 面板密码
- `PALDEFENDER__TOKEN`：上一步生成的 Token
- 共享卷 `./game`：三个服务都挂载它，保证游戏目录与存档一致

#### 4. 启动

```bash
docker compose -f docker-compose.paldefender.yml up -d
docker compose -f docker-compose.paldefender.yml logs -f
```

- 游戏服：UDP `8211`
- PalAdmin 面板：`http://服务器IP:8190`
- PalDefender REST：容器内 `:17993`（PalAdmin 通过 `http://paldefender:17993` 访问）

#### 5. 验证集成

1. 浏览器打开 PalAdmin 面板 →「PalDefender」页，应显示绿色**已连接**
2. 游戏内以管理员身份执行 `/imcheater`，观察 PalDefender 是否响应、PalAdmin 告警页是否出现记录
3. 在「系统设置」确认 `anticheat.mode = integrated`，处置动作会通过 PalDefender 执行（私聊警告、封禁、IP 封禁）

> **注意**：Wine 容器加载 PalDefender 的具体命令/路径与所选 Wine 镜像强相关，
> compose 中的 `command: wine ...` 仅为示例，部署时需对照镜像文档和 PalDefender 安装说明调整。

#### 集成模式 vs 外置模式

| 能力 | 外置模式 (external) | 集成模式 (integrated, +PalDefender) |
|---|---|---|
| 存档属性/复制/非法物品检测 | ✅ | ✅ |
| 在线瞬移/多开检测 | ✅ | ✅ |
| 进程内实时伤害/非法 stat 拦截 | ❌ | ✅ |
| PalDefender 私聊警告/IP 封禁 | ❌ | ✅ |
| 游戏服运行方式 | 原生 Linux | 原生 Linux |
| 是否额外需要 Wine | ❌ | ✅（仅 PalDefender 容器） |

> 两种模式 PalAdmin 都提供面板、告警、审计、回档、Webhook；集成模式多了 PalDefender 的进程内实时防护。

---

### 推送到 Gitee 镜像仓库，服务器直接拉取

不想在服务器上构建镜像时，可把 PalAdmin 镜像推到 **Gitee 容器镜像仓库**，服务器直接拉取运行。

#### 1. 在 Gitee 创建镜像仓库

在 Gitee 新建容器镜像仓库，例如 `qyc-qyc/paladmin`。完整镜像地址形如：

```
gitee.com/qyc-qyc/paladmin:latest
```

> Gitee 镜像仓库的命名空间（用户名/组织）通常**全小写**，请以 Gitee 页面显示的地址为准。

#### 2. 本地构建并推送

脚本已提供，把命名空间改成你的 Gitee 用户名后执行：

```bash
# Linux / macOS
export DOCKER_USERNAME=你的gitee用户名
export DOCKER_PASSWORD=你的Gitee私人令牌   # 在 Gitee 设置→私人令牌生成
bash scripts/docker-push.sh latest
```

```powershell
# Windows PowerShell
$env:DOCKER_PASSWORD="你的Gitee私人令牌"
powershell -ExecutionPolicy Bypass -File scripts\docker-push.ps1
```

脚本会 `docker build --platform linux/amd64` 后 `docker push` 到 Gitee。
（脚本里默认 `qyc-qyc/paladmin`，按需修改 `$NAMESPACE`/`$IMAGE`。）

#### 3. 服务器登录并拉取

```bash
docker login gitee.com                       # 用户名 + Gitee 私人令牌
docker pull gitee.com/qyc-qyc/paladmin:latest
```

#### 4. 用拉取镜像的 compose 启动

- **外置模式**（原生 Linux 游戏服，无 PalDefender）：
  ```bash
  cp .env.example .env && nano .env
  # 把 docker-compose.server.yml 里的镜像地址改成你的
  docker compose -f docker-compose.server.yml up -d
  ```
- **集成模式**（Linux 游戏服 + PalDefender(Wine)）：
  ```bash
  # 把 docker-compose.gitee.yml 里的镜像地址改成你的
  docker compose -f docker-compose.gitee.yml up -d
  ```

这两个 compose 文件里 `paladmin` 服务只有 `image:`、**没有 `build:`**，因此服务器不会本地构建，只拉取 Gitee 上的镜像。更新版本时重新 `docker pull` 后重启即可。

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
  mode: "systemd"           # noop / systemd / docker（回档需要）
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
├── deploy/         # systemd unit
├── scripts/        # 构建与安装脚本
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
```

交叉编译 Linux 二进制（在 Windows 上）：`powershell -File build-linux.ps1`

## 致谢

- [palworld-server-tool](https://github.com/zaigie/palworld-server-tool) —— 存档解析与基础管理框架参考
- [PalDefender](https://github.com/Ultimeit/PalDefender) —— 反作弊设计理念、处置阶梯、REST API 参考
- [palworld-save-tools](https://github.com/cheahjs/palworld-save-tools) —— 存档格式解析
- ID 数据参考 [paldeck.cc](https://paldeck.cc)

## 许可证

本项目采用 MIT 许可证。
