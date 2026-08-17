# PalDefender REST API 对接 PalAdmin 面板方案

> 目标文件：`D:\项目\pal\docs\PalDefender对接方案.md`
> 编写日期：2026-08-17
> 资料来源：`_paldefender/docs/zh/RESTAPI/`、`_paldefender/docs/zh/FileTypes/`
> 适用版本：PalDefender 内置 REST API（base path `/v1/pdapi`，默认端口 `17993`）

---

## 一、PalDefender REST API 概览

### 1.1 连接信息

| 项目 | 值 |
|------|-----|
| 默认 Base URL | `http://127.0.0.1:17993` |
| 默认端口 | `17993` |
| API 前缀 | `/v1/pdapi` |
| 内容类型 | `application/json`（POST 请求） |
| 鉴权方式 | HTTP Header `Authorization: Bearer <token>`（所有端点都需要） |
| 健康检查 | `GET /v1/pdapi/version` |
| 超时特性 | 游戏线程回调 **5 秒** 内未完成返回 `500 REQUEST_TIMEOUT` |

安全提示（官方文档强调）：该 API 设计用于本机/受信任网络，**不要把 17993 端口直接暴露公网**；如需远程访问应放在反向代理（nginx/Caddy/Traefik）后终止 TLS。PalAdmin 后端代理天然满足这一约束——浏览器只访问 PalAdmin，由后端在本机回环调 PalDefender。

### 1.2 配置文件与 API Key 来源

PalDefender 的 REST 配置与其 DLL 同目录，面板已知 Win64 目录路径（来自游戏安装目录推导，见现有 `paldefender.go` 的 `detectAt`）：

```
<游戏安装目录>/Pal/Binaries/Win64/PalDefender/RESTAPI/RESTConfig.json
<游戏安装目录>/Pal/Binaries/Win64/PalDefender/RESTAPI/Tokens/*.json
```

- **`RESTConfig.json`**：控制 `Enabled`（是否启用）、**绑定地址（host/bind）**、**端口（port）**、控制台日志、CORS 设置。首次启动游戏服后生成。
- **`Tokens/*.json`**：每个 `.json` 文件（`TokenExample.json` 除外）都是一个有效令牌文件。结构：
  ```json
  {
    "Name": "AdminPanel",
    "Token": "<随机字符串，即 API key>",
    "Permissions": ["REST.*"]
  }
  ```
- 启动一次服务器会生成示例令牌 `TokenExample.json`；**真正的 API key 需要管理员手动创建**（复制示例文件改名、填入随机 Token、按需收窄 Permissions）。PalDefender 不会自动签发可用令牌。

**API key 的获取策略（推荐组合）：**

1. **首选：用户在面板设置里手动粘贴 Token。** 因为令牌本就应由管理员主动创建、分服务发放，且 `Permissions` 可按最小权限配置（例如只给面板用到的 `REST.Players.Read`、`REST.Punishments.*` 等，而非 `REST.*`）。面板不直接改写 PalDefender 的令牌文件。
2. **可选增强：自动探测/读取。** 面板已知 Win64 路径，可扫描 `PalDefender/RESTAPI/Tokens/` 下除 `TokenExample.json` 外的第一个 `.json`，读出 `Token` 字段作为默认值预填到设置框（仍允许用户覆盖）。若 `RESTConfig.json` 存在，也可解析其 `Enabled`/端口作为“是否已开启 API”的状态提示。
3. host/port 默认 `127.0.0.1:17993`，允许在面板设置中覆盖（极少数部署把 PalDefender 跑在另一台机器/容器）。

### 1.3 玩家标识符（player_identifier）

处罚/奖励/查询端点统一接受 `player_identifier` 路径参数，支持：
- **`UserId`**：平台用户 ID，带平台前缀，如 `steam_76561198012345678`、`gdk_2533274812345678`、`ps5_0f4b8c2d91aa34ef`；
- **`PlayerUID`**：存档用 GUID，如 `b7f4e91a-2c53-4d8f-a6e1-93c4bb62a7d1`；
- 以及其他 PalDefender 支持的标识符。

PalAdmin 现有官方 REST 链路主要使用 `player_uid`（GUID）。对接时应优先用 `PlayerUID` 以保持一致；`GET /players` 返回里同时含 `UserId` 和 `PlayerUID`，可作为映射来源。注意 `unban` 用的是 `user_id`（非 PlayerUID），`banip/unbanip` 用 IP。

### 1.4 通用错误格式

所有错误响应统一结构：

```json
{ "Error": { "Code": "ERROR_CODE", "Message": "人类可读消息", "详情": {} } }
```

通用错误码：`401 INVALID_TOKEN`、`403 MISSING_PERMISSION`、`400 INVALID_JSON`、`400 REQUEST_FAILED`、`500 REQUEST_TIMEOUT`（5 秒）、`400 VALIDATION_FAILED`；各端点另有 `PLAYER_NOT_FOUND`、`BAN_NOT_FOUND`、`BASE_CAMP_NOT_FOUND` 等。后端代理需把这些透传/归一为前端可展示的错误信息。

---

## 二、端点清单（按功能分组）

### 2.1 查询类端点（GET）

| 方法 | 路径 | 权限 | 请求参数 | 返回值 | 用途 |
|------|------|------|----------|--------|------|
| GET | `/v1/pdapi/version` | `REST.Version.Read` | 无 | `Version{Major,Minor,Patch,Build,Version,VersionLong,Beta}` | 健康检查/版本探测，配置后第一个调用的端点 |
| GET | `/v1/pdapi/players` | `REST.Players.Read` | 无 | `Meta{PlayerCount,OnlineCount}` + `Players[]{Name,IP,PlayerUID,UserId,GuildName,GuildUUID,Status,WorldLocation,MapLocation}` | 列出所有已知玩家（含在线/离线、坐标、公会），玩家选择器数据源 |
| GET | `/v1/pdapi/player/<id>` | `REST.Player.Read` | 路径：player_identifier | `Player{...同上单条...}` | 单个玩家详情 |
| GET | `/v1/pdapi/pals/<id>` | `REST.Pals.Read` | 路径：player_identifier | 玩家帕鲁列表 | 查看玩家帕鲁（ID 可查 paldeck.cc/pals） |
| GET | `/v1/pdapi/items/<id>` | `REST.Items.Read` | 路径：player_identifier | 玩家物品列表 | 查看背包（ID 可查 paldeck.cc/items） |
| GET | `/v1/pdapi/techs/<id>` | `REST.Techs.Read` | 路径：player_identifier | 玩家已解锁科技列表 | 查看科技（paldeck.cc/technology） |
| GET | `/v1/pdapi/progression/<id>` | `REST.Progression.Read` | 路径：player_identifier | EXP、等级状态、遗物总数、科技点总数 | 玩家进度概览 |
| GET | `/v1/pdapi/guilds` | `REST.Guilds.Read` | 无 | 公会摘要列表（公会/基地/成员计数） | 公会选择器，找 guild_id |
| GET | `/v1/pdapi/guild/<guild_id>` | `REST.Guild.Read` | 路径：guild_id | 公会详情（成员、基地/营地列表，含 Camp ID） | 删除据点前确认归属 |
| GET | `/v1/pdapi/banlist` | `REST.Banlist.Read` | 查询：`active`(true/false/1)、`entryType`、`userId`、`ip`/`userIP`、`issuerType/issuerName/issuerIP`、`reason`、`q` | `Banlist{Version,BannedMessage,UserEntries[],IPEntries[]}`；条目含 `UserId/IP`、`Active`、`BannedBy/UnbannedBy{Type,NameValue,IP,Reason,Timestamp}` | 封禁记录查询/搜索，数据来自 `Banlist.json` |

### 2.2 处罚类端点（POST）

| 方法 | 路径 | 权限 | 请求体 | 返回值 | 用途 |
|------|------|------|--------|--------|------|
| POST | `/v1/pdapi/kick/<id>` | `REST.Punishments.Kick` | 可选 `{Reason}` | `{Success,UserId}` | 踢出在线玩家，不写封禁记录 |
| POST | `/v1/pdapi/ban/<id>` | `REST.Punishments.Ban` | 可选 `{Reason, IP:bool}` | `{Success,UserId,IP,BannedIP,Kicked}` | 封禁用户并写 `Banlist.json`；`IP:true` 同时封 IP；在线会被踢 |
| POST | `/v1/pdapi/unban/<user_id>` | `REST.Punishments.Unban` | 可选 `{Reason}` | 解封结果 | 解封（注意参数是 **user_id** 非 PlayerUID） |
| POST | `/v1/pdapi/banip/<ip>` | `REST.Punishments.BanIP` | 可选 `{Reason, UserId}` | 封 IP 结果 | 封 IP，可关联 UserId |
| POST | `/v1/pdapi/unbanip/<ip>` | `REST.Punishments.UnbanIP` | 可选 `{Reason}` | 解 IP 结果 | 解封 IP |
| POST | `/v1/pdapi/deletebase/<base_camp_id>` | `REST.Base.Delete` | 可选空对象 `{}` | 删除结果 | 按营地 GUID **删除据点/营地**（破坏性操作，先从 guild 详情确认） |

### 2.3 消息类端点（POST）

| 方法 | 路径 | 权限 | 请求体 | 用途 |
|------|------|------|--------|------|
| POST | `/v1/pdapi/Broadcast` | `REST.Messages.Broadcast` | `{Message}` 必填 | 全服广播聊天消息 |
| POST | `/v1/pdapi/Alert` | `REST.Messages.Alert` | `{Message}` 必填 | 发送**高优先级警报**（区别于普通广播） |
| POST | `/v1/pdapi/SendPlayerMessage` | `REST.Messages.Send.*`（PlayerChat/GlobalChat/GuildChat/Log.Normal/Important/VeryImportant） | `{SendType, Message, UserID}` 或 `{SendType, Message, UserIDs[]}`；二者只能选一个 | 向单人或多人发私聊/公会/日志消息，类型由 `SendType` 决定 |

### 2.4 奖励/工具类端点（POST）

| 方法 | 路径 | 权限 | 请求体 | 用途 |
|------|------|------|--------|------|
| POST | `/v1/pdapi/give/items/<id>` | `REST.Items.Give` | `{Items:[{ItemID,Count}]}` | 给物品（如弹药、金钱），Count 为正数 |
| POST | `/v1/pdapi/give/pals/<id>` | `REST.Pals.Give` | `{Pals:[{PalID,Level}]}` | 按 ID+等级给帕鲁 |
| POST | `/v1/pdapi/give/paleggs/<id>` | `REST.PalEggs.Give` | `{PalEggs:[{EggID, PalID 或 PalTemplate, Level?}]}` | 给帕鲁蛋；`PalID` 与 `PalTemplate` 二选一 |
| POST | `/v1/pdapi/give/paltemplate/<id>` | `REST.PalTemplates.Give` | `{PalTemplates:["xxx.json",...]}` | 按 `Pals/Templates/` 模板文件给自定义帕鲁 |
| POST | `/v1/pdapi/give/progression/<id>` | `REST.Progression.Give` | `{EXP?, TechnologyPoints?, AncientTechnologyPoints?, Relics:{遗物类型:数量}}` | 给经验/科技点/古代点/遗物；遗物类型见下 |
| POST | `/v1/pdapi/learntech/<id>` | `REST.Techs.Learn` | `{Technology: "TechID" 或 ["TechID",...] 或 "All"}` | 学会科技（单个/多个/全部） |
| POST | `/v1/pdapi/forgettech/<id>` | `REST.Techs.Forget` | 同上 | 遗忘科技 |
| POST | `/v1/pdapi/ReloadConfig` | `REST.Reload.Config` | 可选 `{}` | 热重载 PalDefender 配置（Banlist、ImportRules 等），无需重启游戏服 |

**`give/progression` 支持的遗物类型（Relics）：**
`CapturePower`、`HungerReduction`、`SwimSpeed`、`FoodDecayReduction`、`JumpPower`、`GliderSpeed`、`ClimbSpeed`、`StatusAilmentResist`、`StaminaReduction`、`SphereHoming`、`ExpBonus`、`RainbowPassiveRate`、`MoveSpeed`。

> `POST /v1/pdapi/give`（旧版原子奖励）已废弃，不对接。

---

## 三、与 PalAdmin 现有功能的重叠分析

PalAdmin 现有玩家/封禁/广播走的是**官方 Palworld REST API**（`/api/player/...`、`/api/banlist`、`/api/server/broadcast`）。PalDefender 版本能力对比：

| 功能 | 现有官方 REST | PalDefender 版本 | 差异与整合建议 |
|------|---------------|------------------|----------------|
| 玩家列表 | `GET /api/player`、`/api/online_player`（离线/在线分离） | `GET /players` 一次返回全部已知玩家，带 `Status`、IP、公会、坐标、OnlineCount | **并存**。PalDefender 列表信息更丰富（坐标、IP、公会名），可在 PalDefender 页用；现有玩家页保持不动。注意官方 REST 常因游戏版本更新而不稳定，PalDefender 进程内读取更可靠 |
| 踢出 | `POST /api/player/:uid/kick` | `POST /kick/<id>` 支持 Reason | **并存**，PD 版可带原因。不强制替换现有入口 |
| 封禁/解封 | `POST /api/player/:uid/ban`、`/unban` | `POST /ban`（带 Reason、可选连坐 IP、自动踢在线）、`/unban/<user_id>` | **并存**。PD 封禁写 `Banlist.json` 且带原因/执行者/时间戳，更完善。建议封禁页**新增“通过 PalDefender 封禁”选项**，而非替换；两套封禁数据来源不同（官方 banlist vs PalDefender Banlist.json），需在 UI 标注 |
| IP 封禁 | `POST /api/banip`、`/unbanip` | `POST /banip/<ip>`、`/unbanip/<ip>`（带 Reason、可关联 UserId） | **并存**，PD 版审计信息更强 |
| 封禁列表 | `GET /api/banlist?active=true` | `GET /v1/pdapi/banlist`（富查询：userId/ip/reason/q/issuer，含执行者与时间） | **并存**。PD banlist 是独立数据源，建议在封禁页加来源切换 Tab |
| 广播 | `POST /api/server/broadcast` | `POST /Broadcast`，另有 `POST /Alert` 警报、`POST /SendPlayerMessage` 私聊 | **增强并存**。现有广播保留；在 PalDefender 页新增“警报”和“私聊玩家”（官方 REST 无私聊能力） |
| 玩家私聊 | `POST /api/player/:uid/message`（已有 sendPlayerMessage 路由） | `SendPlayerMessage` 支持多人、多类型（聊天/日志/公会） | 可考虑后续用 PD 版替换以获得多人/日志能力，但**第一阶段不动现有路由** |
| 删除据点 | 无 | `POST /deletebase/<camp_id>` | **PalDefender 独有**，高价值新功能 |
| 给物品/帕鲁/蛋/进度 | 无 | give-items/pals/paleggs/paltemplate/progression | **PalDefender 独有**，核心增量功能 |
| 学会/遗忘科技 | 无 | learntech/forgettech | **PalDefender 独有** |
| 热重载配置 | 无（只能重启） | `POST /ReloadConfig` | **PalDefender 独有**，运维便利 |
| 版本/健康 | 无独立 PD 探测 | `GET /version` | 用于配置连通性测试 |

**整合原则：** 第一阶段**不替换任何现有官方 REST 功能**，PalDefender 能力作为“增强工具”集中在 PalDefender 页内；待验证稳定后，再在封禁/广播页提供“使用 PalDefender”的可选开关。这样避免破坏现有可用链路，也规避两套封禁数据混淆。

---

## 四、后端对接架构

### 4.1 推荐：Go 后端代理（不采用前端直连）

| 维度 | 后端代理（推荐） | 前端直连 PalDefender |
|------|------------------|----------------------|
| 鉴权 | 浏览器只持 PalAdmin JWT；PD Token 存后端，不暴露 | Token 会下发到浏览器，易泄露 |
| 跨域 | 无（后端同源回环） | 需 PalDefender 配 CORS，且 PD 默认绑定 localhost |
| 网络暴露 | PD 端口只监听 127.0.0.1，无需对公网/面板用户开放 | 必须让浏览器能访问 PD 端口，违反官方安全建议 |
| 错误归一 | 后端统一把 PD 错误码转成 `{error}` 格式 | 前端需处理两套错误格式 |
| 审计 | 可在后端记录管理员操作（现有 audit 服务） | 难统一审计 |
| 配置复用 | 后端已有 settings/viper/游戏目录推导 | 前端需重复管理地址与 Token |

**结论：在 PalAdmin Go 后端新增 PalDefender HTTP client，所有 PD 调用由 `/api/paldefender/*` 代理；前端只与 PalAdmin 同源通信。**

### 4.2 新增/改动后端文件

| 文件 | 职责 |
|------|------|
| `internal/paldefender/client.go`（新增） | PalDefender REST client：从 settings 读取 host/port/token；封装 `do(method, path, body)`，统一注入 `Authorization: Bearer`、`Content-Type`、10 秒超时（PD 内部 5 秒）、解析 `Error.Code` 并返回结构化错误；提供 `Version()` 等方法 |
| `api/paldefender_api.go`（新增，或扩展现有 `paldefender.go`） | Gin handler：每个对接端点对应一个 handler，调用 client；破坏性操作（ban/deletebase/give-*）写 `audit` 日志；保留现有 `paldefender.go` 的安装/Wine 部分不动 |
| `service/settings.go`（修改） | 新增 PalDefender 相关 Setting 常量与默认值（见第六节） |
| `api/settings.go`（修改） | 在 `isEditableKey` 白名单加入新键；`isSecret` 已对含 `token`/`password` 的键脱敏，新 token 键命名需含 `token` 以自动脱敏 |
| `api/router.go`（修改） | 在现有 `authGroup.GET/POST("/paldefender/...")` 块下注册新路由（见 4.3） |

> 注意：现有 `paldefender.go` 里的 `palDefenderAPI` 结构体已注册在 router 中。建议把“安装管理”和“REST API 代理”拆成两个文件但同属 `api` 包，或在同一结构体内增加方法，保持路由集中。

### 4.3 路由设计（`/api/paldefender/*`，全部走 JWT 鉴权组）

现有安装路由保持不变，新增：

```
# 连通性
GET  /api/paldefender/api/version              # 调 GET /version，测试连接

# 玩家
GET  /api/paldefender/players                  # GET /players
GET  /api/paldefender/players/:id              # GET /player/<id>
GET  /api/paldefender/players/:id/pals         # GET /pals/<id>
GET  /api/paldefender/players/:id/items        # GET /items/<id>
GET  /api/paldefender/players/:id/techs        # GET /techs/<id>
GET  /api/paldefender/players/:id/progression  # GET /progression/<id>

# 公会/据点
GET  /api/paldefender/guilds                   # GET /guilds
GET  /api/paldefender/guilds/:gid              # GET /guild/<gid>
POST /api/paldefender/bases/:cid/delete        # POST /deletebase/<cid>

# 封禁
GET  /api/paldefender/banlist                  # GET /banlist（透传 query）
POST /api/paldefender/players/:id/kick         # POST /kick/<id>
POST /api/paldefender/players/:id/ban          # POST /ban/<id>
POST /api/paldefender/unban/:user_id           # POST /unban/<user_id>
POST /api/paldefender/banip/:ip                # POST /banip/<ip>
POST /api/paldefender/unbanip/:ip              # POST /unbanip/<ip>

# 消息
POST /api/paldefender/broadcast                # POST /Broadcast
POST /api/paldefender/alert                    # POST /Alert
POST /api/paldefender/message                  # POST /SendPlayerMessage

# 奖励/工具
POST /api/paldefender/players/:id/give-items       # POST /give/items/<id>
POST /api/paldefender/players/:id/give-pals        # POST /give/pals/<id>
POST /api/paldefender/players/:id/give-paleggs     # POST /give/paleggs/<id>
POST /api/paldefender/players/:id/give-paltemplate # POST /give/paltemplate/<id>
POST /api/paldefender/players/:id/give-progression # POST /give/progression/<id>
POST /api/paldefender/players/:id/learntech        # POST /learntech/<id>
POST /api/paldefender/players/:id/forgettech       # POST /forgettech/<id>
POST /api/paldefender/reload-config                # POST /ReloadConfig
```

设计要点：
- 使用 PalAdmin 风格的路径参数（`:id`），body 透传 PD 所需 JSON 字段；handler 内部拼 PD URL。
- `:id` 允许含前缀（`steam_xxx`）和 GUID，Gin 默认参数不含 `/`，可用；如遇特殊字符再做 URL encode。
- `banlist` 透传 query（`active/userId/ip/q/...`）到 PD。
- 破坏性/写操作（kick/ban/banip/deletebase/give-*/learntech/forgettech/reload）统一在 handler 调 `audit.Add(...)`，记录操作者、目标、原因。
- 后端代理时若 settings 中 `paldefender.enabled` 为 false 或 token 为空，直接返回明确错误（“PalDefender API 未配置”）。

---

## 五、前端对接与 UI 规划

### 5.1 API 封装

在 `web/src/api/index.ts` 的 `api` 对象中新增 `paldefender.*` 方法，沿用现有 `request<T>` 封装（自动带 JWT、统一错误抛出）。例如 `pdVersion()`、`pdGetPlayers()`、`pdKick(id, body)`、`pdGiveItems(id, items)` 等。不要在前端硬编码 PD 地址/token。

### 5.2 页面结构：不新增顶级菜单，全部收纳在 PalDefender 页

现有菜单已有“PalDefender”项（`Layout.vue` 的 `menuOptions`，路由 `/paldefender`）。建议**保留单页、用 `n-tabs` 分区**，避免菜单膨胀：

1. **安装与状态（现有区域，保留）**：Wine 状态、DLL 安装/卸载、进度弹窗。在卡片顶部新增一个“REST API 连接状态”条：调 `/api/paldefender/api/version`，显示 PD 版本号 + 在线/未配置/连接失败，并提供“测试连接”按钮。
2. **玩家管理（新）**：表格展示 `GET /players`（名称、UserId、PlayerUID、IP、公会、在线状态、坐标）；行操作：查看详情（帕鲁/物品/科技/进度抽屉或子页）、私聊、踢出、封禁。支持按名称/UserId 搜索。
3. **封禁列表（新）**：展示 `GET /banlist`，Tab 切换 用户封禁/IP 封禁；支持搜索（q/userId/ip/reason）、解封/解 IP；提供“手动封 IP”入口。**标注数据来源为 PalDefender Banlist.json**，与现有顶级“封禁”页（官方 REST）区分。
4. **广播与警报（新）**：
   - 广播输入框 → `/broadcast`；
   - 警报输入框 → `/alert`（红色样式，提示高优先级）；
   - 私聊：选择玩家（多选）+ SendType 下拉 + 消息 → `/message`。
5. **公会与据点（新）**：公会列表 → 查看详情（成员、营地列表，显示 Camp ID）→ 每个据点带“删除据点”按钮（二次确认，红色，输入营地 ID 或勾选确认）。
6. **管理工具（新，分折叠面板）**：
   - 给物品：选玩家 + 动态物品行（ItemID + Count，可增删）→ give-items；
   - 给帕鲁：PalID + Level；
   - 给帕鲁蛋：EggID + PalID/PalTemplate；
   - 给模板帕鲁：模板文件名列表；
   - 给进度：EXP / 科技点 / 古代点 / 遗物数量表单；
   - 学会/遗忘科技：TechID 或“全部”；
   - 热重载配置按钮（二次确认）。

### 5.3 安全与交互细节

- 所有破坏性操作（ban、deletebase、give/progression、learn/forget “All”、reload）使用 `n-popconfirm` 二次确认；deletebase 与“学会全部/遗忘全部”额外要求输入确认文字或勾选。
- 原因（Reason）字段在 kick/ban/banip 表单中提供，便于审计。
- ItemID/PalID/TechID 输入给出 paldeck.cc 链接提示（ID 而非显示名）。
- Token 在设置页脱敏（后端已对含 `token` 键返回 `********`）。

---

## 六、设置项设计

在 `service/settings.go` 新增常量，存入现有 bbolt settings bucket（与 rest/rcon 同套机制），前端“设置”页新增“PalDefender”分组：

| Setting Key | 含义 | 默认值 | 说明 |
|-------------|------|--------|------|
| `paldefender.enabled` | 是否启用 PalDefender REST 对接 | `false` | 关闭时后端代理直接拒绝并提示未启用 |
| `paldefender.host` | PalDefender API 主机 | `127.0.0.1` | 一般为本机回环 |
| `paldefender.port` | API 端口 | `17993` | 来自 RESTConfig.json |
| `paldefender.token` | Bearer Token（API key） | 空 | 含 `token`，自动被 `isSecret` 脱敏；用户从 Tokens/*.json 粘贴 |
| `paldefender.base_path` | API 前缀（可选） | `/v1/pdapi` | 高级项，一般不用改，可不在 UI 暴露 |

后端拼装 Base URL：`http://{host}:{port}{base_path}`。

需要在 `api/settings.go` 的 `isEditableKey` 中把上述键加入白名单（`paldefender.base_path` 若不暴露则不加入）。`paldefender.token` 因含 `token` 子串已被 `isSecret` 识别为敏感字段，保存时 `********` 占位符不会覆盖真值——复用现有逻辑即可。

**可选增强（非必须）：** 后端新增一个“自动探测”端点 `/api/paldefender/detect-api`：根据现有 `detectAt(gameDir)` 得到的 Win64 路径，读取 `PalDefender/RESTConfig/RESTConfig.json`（Enabled/host/port）和扫描 `Tokens/*.json` 取首个非示例令牌，回填到设置表单。这能降低配置门槛，但需注意：令牌文件读取意味着后端能看到明文 Token，仅在本机部署且文件权限受限时可接受。

---

## 七、分阶段实施建议

### 第一阶段（最小可用，1 个迭代）

目标：跑通代理链路 + 高频管理能力。

1. 后端：`internal/paldefender/client.go`（Base URL + Token + 通用 do/错误解析）；settings 新增 4 个键；注册 `/api/paldefender/api/version` 与设置页“测试连接”。
2. 高优先级端点代理：`players` 列表、`kick`、`ban/unban`、`banip/unbanip`、`broadcast`、`alert`、`banlist`、`send-player-message`、`deletebase`、`reload-config`、`version`。
3. 前端：PalDefender.vue 用 tabs 改造，保留安装区；新增“连接状态条”、“玩家管理”（列表+踢/封/私聊）、“封禁列表”、“广播与警报”、“公会与据点（仅列表+删除据点）”、“热重载”。
4. 写操作接入 audit 日志。
5. 不改动任何现有官方 REST 路由与页面。

### 第二阶段（强力工具）

1. 奖励类：give-items、give-pals、give-paleggs、give-paltemplate、give-progression。
2. 科技类：learntech、forgettech。
3. 玩家详情抽屉：pals、items、techs、progression 查询。
4. 前端表单完善：物品/帕鲁多行编辑、遗物勾选、“全部科技”危险确认。
5. （可选）在现有顶级“封禁”页增加“通过 PalDefender 操作”的来源切换，把 PD banlist 与官方 banlist 并排展示。

### 第三阶段（打磨与整合）

1. 自动探测 RESTConfig/Token 文件，一键回填设置。
2. 玩家列表定时刷新、在线状态高亮、坐标跳转玩家地图（复用现有 playermap）。
3. 模板帕鲁文件管理（列出 `Pals/Templates/`，未来可扩展上传）。
4. 视稳定情况，将广播/私聊入口在官方 REST 不可用时回退到 PalDefender，或提供系统级“优先使用 PalDefender”开关。
5. 更细粒度的 PalDefender Token 权限建议（在文档/UI 提示管理员按最小权限创建 Token）。

---

## 八、关键注意事项

1. **5 秒超时**：PD 游戏线程回调 5 秒不完成即返回 `REQUEST_TIMEOUT`。后端 client 超时设为 10 秒留余量；give-* 等大批量操作可能触发，前端应提示分批操作。
2. **封禁数据双源**：官方 banlist 与 PalDefender `Banlist.json` 不互通，UI 必须标注来源，避免管理员误判。
3. **标识符差异**：`ban`/`kick`/查询用 `player_identifier`（UserId 或 PlayerUID 均可），但 `unban` 只认 `user_id`；前端玩家行需同时保存 UserId 与 PlayerUID，解封时用 UserId。
4. **Token 权限**：建议管理员创建面板专用 Token 时按需授权（如 `REST.Players.Read`、`REST.Punishments.*`、`REST.Messages.*`、`REST.Items.Give`、`REST.Pals.Give`、`REST.Techs.*`、`REST.Base.Delete`、`REST.Reload.Config`、`REST.Version.Read`），而非一律 `REST.*`。
5. **deletebase 不可逆**：必须先经 `guilds → guild` 确认营地归属与 Camp ID，二次确认后方可调用。
6. **learntech/forgettech 的 “All”**：是裸字符串而非数组，前后端构造 body 时注意区分。
7. **PalDefender 未安装/未启用 API 时**：安装状态区与 API 功能区解耦——DLL 已装但 RESTConfig `Enabled=false` 时，连接状态应明确提示“需在 RESTConfig.json 启用 API 并重启游戏服”。
8. **Wine/进程级依赖**：PalDefender 是 DLL，仅在通过 Wine 启动游戏服并成功加载后，其 REST API 才在线。版本探测失败时，UI 应提示检查游戏服是否以 Wine 启动、DLL 是否加载、RESTConfig 是否启用。

---

## 附：现有代码现状速览

- `api/paldefender.go`：仅实现安装状态检测（Wine/d3d9.dll/PalDefender.dll）、Wine 一键安装、DLL 下载安装/卸载、安装进度轮询。**无任何 REST API 调用**。
- `web/src/views/PalDefender.vue`：上述安装能力的 UI，三个卡片（状态/Wine/插件）+ 两个进度弹窗。
- `web/src/api/index.ts`：统一 `request` 封装，Bearer JWT；现有玩家/封禁/广播走官方 REST。
- `api/router.go`：`/api/paldefender/*` 路由注册在 JWT 鉴权组内，安装类 8 条路由已就位，可直接在其下方追加代理路由。
- `service/settings.go` + `api/settings.go`：bbolt 存储的动态设置，`isEditableKey` 白名单 + `isSecret`（含 password/token 脱敏），新增 PD 设置直接复用。
- PalDefender 配置项**不在**游戏服 `.ini`/palconfig schema 中；其配置位于 Win64/PalDefender 目录，与游戏配置分离。
