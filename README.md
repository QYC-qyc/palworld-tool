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

目标系统：**Ubuntu 22.04**（其他 Linux 发行版同理）。全部基于**代码部署**（从 Gitee 克隆源码后在服务器构建/运行），不使用预构建镜像。

- **方式 A：二进制 + systemd**——直接在服务器编译运行，最简单
- **方式 B：Docker**——用 Docker 多阶段构建，适合游戏服也容器化
- **方式 C：加 PalDefender 实时防护**——Linux 游戏服 + Wine 容器运行 PalDefender

> 游戏服需在 `PalWorldSettings.ini` 开启：`RESTAPIEnabled=True`、`RESTAPIPort=8212`、`RCONEnabled=True`、`RCONPort=25575`，并设置 `AdminPassword`。

代码仓库：`https://gitee.com/QYC-qyc/palworld-tool.git`

---

### 方式 A：二进制 + systemd（服务器直接编译）

#### 1. 安装编译依赖

```bash
sudo apt update
sudo apt install -y golang-go nodejs npm python3 python3-pip git
```

> Go 需 1.22+。若 apt 源版本较旧，从 https://go.dev/dl/ 安装新版。

#### 2. 克隆代码并构建

```bash
git clone https://gitee.com/QYC-qyc/palworld-tool.git paladmin
cd paladmin
bash scripts/build.sh          # 编译后端 + 前端，产出 dist/paladmin/
cd dist/paladmin
```

#### 3. 安装存档解析依赖

```bash
sudo pip3 install -r module/requirements.txt --break-system-packages \
  || sudo pip3 install -r module/requirements.txt
```

#### 4. 修改配置

```bash
cp config.example.yaml config.yaml
nano config.yaml
```

至少确认：

```yaml
web:
  password: "强密码"
  port: 8190
rest:
  address: "http://127.0.0.1:8212"
  password: "游戏AdminPassword"
rcon:
  address: "127.0.0.1:25575"
save:
  # find / -name Level.sav 2>/dev/null 查找，取所在目录
  path: "/home/steam/Pal/Saved/SaveGames/0/<GUID>"
process:
  mode: "systemd"          # 回档需要，填游戏服务名
  service: "palworld"
```

#### 5. 注册并启动服务

```bash
sudo bash install.sh       # 创建 paladmin 用户、注册 systemd、开机自启
sudo systemctl status paladmin
sudo journalctl -u paladmin -f
```

#### 6. 验证

```bash
curl http://127.0.0.1:8190/health    # 期望 {"status":"ok"}
```

浏览器打开 `http://服务器IP:8190` 登录。

---

### 方式 B：Docker（源码构建）

前提：服务器已安装 Docker。在服务器克隆代码后用 Dockerfile 现场构建 PalAdmin 镜像。

```bash
git clone https://gitee.com/QYC-qyc/palworld-tool.git paladmin
cd paladmin

cp .env.example .env
nano .env                 # 设置 WEB_PASSWORD、GAME_ADMIN_PASSWORD
nano docker-compose.yml   # 把存档挂载改成真实路径
```

`docker-compose.yml` 中 `paladmin` 服务使用 `build: .`（从源码构建），关键配置：

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
  PROCESS__CONTAINER: "palworld"
```

启动（`--build` 在服务器构建镜像）：

```bash
docker compose up -d --build
docker compose logs -f paladmin
```

访问 `http://服务器IP:8190`。

---

### 方式 C：加 PalDefender 实时防护（Linux 游戏服 + Wine 容器）

> **⚠️ 关于 PalDefender**
>
> [PalDefender](https://github.com/Ultimeit/PalDefender) 是 Windows DLL，**只有它需要 Wine**，游戏服本身仍是原生 Linux。三服务架构：
>
> | 服务 | 运行方式 | 作用 |
> |---|---|---|
> | `palworld` | 原生 Linux 镜像 | 游戏服务器 |
> | `paldefender` | Wine 容器 | 运行 Windows 版 PalDefender |
> | `paladmin` | 从源码构建 | 面板 + 反作弊，集成模式对接 |

使用 `docker-compose.paldefender.yml`（paladmin 用 `build: .` 从代码构建）：

```bash
git clone https://gitee.com/QYC-qyc/palworld-tool.git paladmin
cd paladmin
cp .env.example .env
nano .env   # WEB_PASSWORD、GAME_ADMIN_PASSWORD、PALDEFENDER_TOKEN
```

#### 1. 准备 PalDefender 文件

从 [PalDefender Releases](https://github.com/Ultimeit/PalDefender/releases) 下载 `PalDefender_Windows.zip`，解压为：

```
./PalDefender/
├── PalDefender.dll
├── d3d9.dll
├── RESTAPI/Tokens/paladmin.json
└── ...
```

`paladmin.json` 内容：

```json
{
  "Name": "PalAdmin",
  "Token": "一段随机长字符串（与 .env 的 PALDEFENDER_TOKEN 一致）",
  "Permissions": ["REST.*"]
}
```

#### 2. 核对 compose

```bash
nano docker-compose.paldefender.yml
```

- `palworld` 用原生 Linux 游戏服镜像
- `paldefender` 用 Wine 镜像，`command: wine ...` 的路径依镜像调整
- 三个服务共享 `./game` 卷

#### 3. 启动

```bash
docker compose -f docker-compose.paldefender.yml up -d --build
docker compose -f docker-compose.paldefender.yml logs -f
```

- 游戏服：UDP `8211`
- 面板：`http://服务器IP:8190`
- PalDefender REST：容器内 `:17993`

#### 4. 验证集成

1. 面板 →「PalDefender」页显示绿色**已连接**
2. 游戏内执行 `/imcheater`，观察面板告警
3. 「系统设置」确认 `anticheat.mode = integrated`

> Wine 加载 PalDefender 的命令/路径与所选镜像强相关，compose 中的 `command` 为示例，需对照镜像文档调整。

#### 集成模式 vs 外置模式

| 能力 | 外置 (方式 B) | 集成 (方式 C) |
|---|---|---|
| 存档属性/复制/非法物品检测 | ✅ | ✅ |
| 在线瞬移/多开检测 | ✅ | ✅ |
| 进程内实时伤害/非法 stat 拦截 | ❌ | ✅ |
| PalDefender 私聊警告/IP 封禁 | ❌ | ✅ |
| 游戏服运行方式 | 原生 Linux | 原生 Linux |
| 是否额外需要 Wine | ❌ | ✅（仅 PalDefender 容器） |

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

```bash
# 进入代码目录拉取最新代码
cd paladmin && git pull

# 二进制（systemd）：重新构建并重启
bash scripts/build.sh && sudo systemctl restart paladmin

# Docker（源码构建）：
docker compose up -d --build
# PalDefender 集成模式：
docker compose -f docker-compose.paldefender.yml up -d --build
```

更多说明见 [DEPLOY.md](DEPLOY.md)。

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
