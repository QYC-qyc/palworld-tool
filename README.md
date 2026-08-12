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

## 快速开始

### Docker（推荐）

```bash
git clone https://gitee.com/QYC-qyc/palworld-tool.git paladmin
cd paladmin
cp .env.example .env       # 编辑密码
# 修改 docker-compose.yml 中的存档挂载路径
docker compose up -d --build
```

访问 `http://服务器IP:8190`，默认密码见 `config.yaml`（首次登录后请立即在「系统设置」修改）。

### 二进制（systemd）

从 [Releases](../../releases) 下载 `paladmin-linux-amd64.tar.gz`，或自行构建：

```bash
# 构建机需 Go 1.22+、Node 18+、Python 3
bash scripts/build.sh          # 产出 dist/paladmin-*-linux-amd64.tar.gz

# 目标服务器
tar -xzf paladmin-*-linux-amd64.tar.gz -C /opt && cd /opt/paladmin
cp config.example.yaml config.yaml   # 修改配置
sudo apt install -y python3 python3-pip
sudo pip3 install -r module/requirements.txt
sudo bash install.sh         # 注册 systemd 服务
```

详细部署见 [DEPLOY.md](DEPLOY.md)，上线前请对照 [docs/联调清单.md](docs/联调清单.md)。

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
