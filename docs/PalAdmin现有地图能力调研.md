# PalAdmin 现有地图能力调研

> 调研日期：2026-08-17
> 目的：为地图功能重写摸清后端数据能力、存档解析现状、地图瓦片与前端实现，明确可复用项与缺失项。
> 所有结论均基于实际读取的源码文件，路径为项目内绝对或相对路径。

---

## 1. 在线玩家 API

### 1.1 前端接口封装（`web/src/api/index.ts`）

与地图/玩家相关的已封装方法：

| 方法 | HTTP | 路径 | 说明 |
|------|------|------|------|
| `api.getOnline()` | GET | `/api/online_player` | 在线玩家（实时 REST） |
| `api.getPlayers()` | GET | `/api/player` | 全量玩家列表（存档，精简结构） |
| `api.getPlayer(uid)` | GET | `/api/player/:player_uid` | 单个玩家详情（含帕鲁/物品） |
| `api.getGuilds()` | GET | `/api/guild` | 公会列表 |
| `api.getGuild(uid)` | GET | `/api/guild/:admin_player_uid` | 单个公会 |
| `api.kickPlayer(uid)` | POST | `/api/player/:uid/kick` | 踢出 |
| `api.banPlayer(uid)` | POST | `/api/player/:uid/ban` | 封禁 |
| `api.sync(from)` | POST | `/api/sync?from=all` | 触发同步 |

`web/src/api/` 目录下另有 `gameserver.ts`、`gamesettings.ts`，与地图无关。

### 1.2 后端路由（`api/router.go`）

匿名可读（无需登录）的玩家/公会路由：

```go
anon.GET("/player", listPlayers)
anon.GET("/player/:player_uid", getPlayer)
anon.GET("/online_player", listOnlinePlayers)
anon.GET("/guild", listGuilds)
anon.GET("/guild/:admin_player_uid", getGuild)
```

需要鉴权的写操作：`PUT /player`、`PUT /guild`（sav_cli 回调用）、`POST /player/:uid/kick|ban|unban|ipban|message`、`POST /sync`。

### 1.3 `ShowPlayers()` 返回字段（`internal/tool/rest_api.go`）

官方 REST `/v1/api/players` 的响应结构：

```go
type ResponsePlayer struct {
    Name      string  `json:"name"`
    PlayerId  string  `json:"playerId"`
    UserId    string  `json:"userId"`
    Ip        string  `json:"ip"`
    Ping      float64 `json:"ping"`
    LocationX float64 `json:"location_x"`
    LocationY float64 `json:"location_y"`
    Level     int     `json:"level"`
}
```

`ShowPlayers()` 把它转换成 `[]database.OnlinePlayer`：

```go
type OnlinePlayer struct {
    PlayerUid  string    `json:"player_uid"`   // 由 playerId 前8位hex转十进制
    SteamId    string    `json:"steam_id"`     // 由 userId 去掉 "steam_" 前缀
    Nickname   string    `json:"nickname"`
    Ip         string    `json:"ip"`
    Ping       float64   `json:"ping"`
    LocationX  float64   `json:"location_x"`
    LocationY  float64   `json:"location_y"`
    Level      int32     `json:"level"`
    LastOnline time.Time `json:"last_online"`
}
```

字段齐全：**location_x / location_y / level / ping / nickname / player_uid 全部有**。注意坐标是游戏世界坐标（Palworld 单位，范围约 ±百万级），需要前端做坐标变换才能上图。

`playerId` 形如 `XXXXXXXX000000000000000000000000`，前 8 位 hex 转十进制得到 `player_uid`；`userId` 形如 `steam_<decimal>`，去掉前缀得到 `steam_id`。

### 1.4 玩家相关 handler（`api/player.go`）

- `listOnlinePlayers`：**直接调用 `tool.ShowPlayers()`**，出错时返回空数组 `[]`，不读数据库。
- `listPlayers`：调 `service.ListPlayers(db)`，从 bbolt 读存档解析出的精简玩家列表。
- `getPlayer`：调 `service.GetPlayer(db, uid)`，读完整玩家（含帕鲁/物品）。
- `putPlayers`：接收 sav_cli PUT 上来的 `[]database.Player`，写库后异步触发反作弊检测。
- `kickPlayer/banPlayer/...`：通过 `tool.KickPlayer` 等调用官方 REST。

关键点：**在线坐标只在玩家在线时可用**；离线玩家的坐标不会被存档解析保留（见第 2 节）。

---

## 2. 玩家数据

### 2.1 全量玩家列表 API

有。`GET /api/player` → `service.ListPlayers()`（`service/player.go`）：

```go
func ListPlayers(db *bbolt.DB) ([]database.TersePlayer, error)
```

从 bbolt 的 `players` bucket 遍历，返回 `[]TersePlayer`。数据来源是 sav_cli 定时解析 Level.sav 后通过 `PUT /api/player` 回写。

### 2.2 玩家数据结构（`internal/database/models.go`）

```go
type TersePlayer struct {
    PlayerUid      string           `json:"player_uid"`
    Nickname       string           `json:"nickname"`
    PlatformID     string           `json:"platform_id,omitempty"`
    Level          int32            `json:"level"`
    Exp            int64            `json:"exp"`
    Hp             int64            `json:"hp"`
    MaxHp          int64            `json:"max_hp"`
    ShieldHp       int64            `json:"shield_hp"`
    ShieldMaxHp    int64            `json:"shield_max_hp"`
    MaxStatusPoint int32            `json:"max_status_point"`
    StatusPoint    map[string]int32 `json:"status_point"`
    FullStomach    float64          `json:"full_stomach"`
    SaveLastOnline string           `json:"save_last_online"`
    OnlinePlayer                     // 嵌入：含 LocationX/LocationY/Ping/Ip/LastOnline
}
```

`TersePlayer` 嵌入了 `OnlinePlayer`，因此**结构上有 `location_x/location_y` 字段**，但这些字段是在线时由 REST 写入并在 `PutPlayers` 中被保留的（见 `service/player.go` 第 28-38 行，存档回写时会保留旧的 `LocationX/LocationY/Ping/Ip/LastOnline`）。也就是说：

- 在线玩家：坐标实时准确。
- 离线玩家：坐标是「最后一次在线时」的快照，由定时任务 `SyncPlayersOnce`（默认 60s）通过 REST 刷新并写库；若玩家在两次同步间隔内下线，坐标停留在最后一次轮询值。
- **存档本身（Level.sav 的 CharacterSaveParameterMap）并未解析玩家坐标**，`module/world_types.py` 的 `Player` 类没有坐标字段。

完整玩家 `Player` 结构在 `TersePlayer` 基础上增加 `Pals []*Pal` 和 `Items *Items`。

### 2.3 存档解析方式

采用**外部 Python 程序 sav_cli**，不是 Go 自己解析。

- 入口：`internal/tool/save.go` 的 `Decode(path)`：
  ```go
  func Decode(path string) error
  ```
  逻辑：找到 `sav_cli` 可执行文件（配置 `save.decode_path`，默认工作目录下 `sav_cli` / `sav_cli.exe`），把 Level.sav 交给它，sav_cli 解析后通过 HTTP PUT 回写到 `http://127.0.0.1:<port>/api/player` 和 `/api/guild`（带 JWT token）。
- sav_cli 源码位于 `module/`（Python）：
  - `module/sav_cli.py`：入口，参数 `-f <Level.sav> --request <url> --token <jwt>`。
  - `module/structurer.py`：核心解析，调用 `palworld-save-tools` 库。
  - `module/world_types.py`：Player / Pal / Guild 数据类。
  - `module/logger.py`、`module/requirements.txt`（`palworld-save-tools>=0.20.0`、`requests`）。
- 部署：`Dockerfile` 用 PyInstaller 把 `module/` 打包成单文件 `sav_cli`，通过环境变量 `SAVE__DECODE_PATH=/app/sav_cli` 指定。`scripts/install.sh` 也会释放 `sav_cli` 二进制并写配置。
- 定时调度：`internal/task/task.go` 的 `Schedule()`，按 `save.sync_interval`（默认 120s）调 `SyncSavOnce()` → `tool.Decode(EffectiveSavePath())`。在线玩家同步按 `task.sync_interval`（默认 60s）调 `SyncPlayersOnce()` → `tool.ShowPlayers()`。

`module/structurer.py` 解析了以下 Level.sav 节点：

| worldSaveData 节点 | 是否解析 | 产出 |
|---|---|---|
| `CharacterSaveParameterMap` | 是 | 玩家列表、帕鲁列表（含玩家物品容器关联） |
| `ItemContainerSaveData` | 是（含 skip-decode 后再解析） | 玩家 6 类容器物品 |
| `GroupSaveDataMap` | 是（过滤 `EPalGroupType::Guild`） | 公会列表 |
| `BaseCampSaveData` | **否** | 无 |
| `MapObjectSaveData` / `FoliageGridSaveDataMap` 等 | skip（跳过不解析） | 无 |

玩家物品还会读 `Players/<UUID>.sav` 单玩家存档。

---

## 3. 公会数据（重点）

### 3.1 公会列表 API

有。`GET /api/guild` → `service.ListGuilds(db)`（`service/guild.go`），返回 `[]database.Guild`。数据由 sav_cli `PUT /api/guild` 写入 bbolt。

### 3.2 公会数据结构（`internal/database/models.go`）

```go
type Guild struct {
    Name           string         `json:"name"`
    BaseCampLevel  int32          `json:"base_camp_level"`
    AdminPlayerUid string         `json:"admin_player_uid"`
    Players        []*GuildPlayer `json:"players"`
    BaseIds        []string       `json:"base_ids"`
}

type GuildPlayer struct {
    PlayerUid string `json:"player_uid"`
    Nickname  string `json:"nickname"`
}
```

**结论：当前 Go 端 Guild 结构没有 `base_camp` 字段**，只有 `base_ids`（据点 ID 字符串列表），没有据点坐标、area、location。

### 3.3 是否解析 BaseCampSaveData

**没有。** 本项目的 `module/structurer.py` 的 `structure_guild()` 只读 `GroupSaveDataMap`：

```python
def structure_guild(filetime: int = -1):
    if not wsd.get("GroupSaveDataMap"):
        return []
    groups = (
        g["value"]["RawData"]["value"]
        for g in wsd["GroupSaveDataMap"]["value"]
        if g["value"]["GroupType"]["value"]["value"] == "EPalGroupType::Guild"
    )
    Ticks = wsd["GameTimeSaveData"]["value"]["RealDateTimeTicks"]["value"]
    guilds_generator = (Guild(g, Ticks, filetime).to_dict() for g in groups)
    ...
```

`module/world_types.py` 的 `Guild` 类只输出 `name / base_camp_level / admin_player_uid / players / base_ids`，没有 `base_camp`，也没有 `BaseCamp` 类。

存档中 `worldSaveData.BaseCampSaveData` 在 `SKP_PALWORLD_CUSTOM_PROPERTIES` 里未被列入 skip 名单，但 `structurer.py` 全文没有引用它，因此**据点数据完全未被读取**。

### 3.4 参考项目 `_palworld-server-tool/` 的 sav_cli 能否复用

参考项目（`_palworld-server-tool/sav_cli/`）**已经实现了据点解析**，可作为移植蓝本：

`_palworld-server-tool/sav_cli/structurer.py`：

```python
def structure_base_camp():
    if not wsd.get("BaseCampSaveData"):
        return []
    return [
        BaseCamp(b["value"]["RawData"]["value"]).to_dict()
        for b in wsd["BaseCampSaveData"]["value"]
    ]

def structure_guild(filetime: int = -1):
    ...
    base_camps = structure_base_camp()
    ...
    for guild in sorted_guilds:
        for camp in base_camps:
            if camp["id"] in guild["base_ids"]:
                guild["base_camp"].append({
                    "id": camp["id"],
                    "area": camp["area_range"],
                    "location_x": camp["transform"]["x"],
                    "location_y": camp["transform"]["y"],
                })
    return list(sorted_guilds)
```

`_palworld-server-tool/sav_cli/world_types.py` 定义了 `BaseCamp` 类和带 `base_camp=[]` 的 `Guild`：

```python
class BaseCamp:
    def __init__(self, data):
        self.id = hexuid_to_decimal(data["id"])
        self.state = data["state"]
        self.transform = {
            "x": data["transform"]["translation"]["x"],
            "y": data["transform"]["translation"]["y"],
            "z": data["transform"]["translation"]["z"],
            "rotation": {...},
        }
        self.area_range = data["area_range"]
        self.group_id_belong_to = hexuid_to_decimal(data["group_id_belong_to"])
        self.owner_map_object_instance_id = hexuid_to_decimal(...)

class Guild:
    ...
    self.base_camp = []
    __order = [..., "base_ids", "base_camp"]
```

参考项目 Go 端（`_palworld-server-tool/internal/database/models.go`）也有对应结构：

```go
type BaseCamp struct {
    Id        string  `json:"id"`
    Area      float64 `json:"area"`
    LocationX float64 `json:"location_x"`
    LocationY float64 `json:"location_y"`
}
type Guild struct {
    ...
    BaseCamp []BaseCamp `json:"base_camp"`
}
```

**复用判断：**

- 本项目 sav_cli 是 Python，位于 `module/`，与参考项目语言一致，可以直接移植 `structure_base_camp()` 和 `BaseCamp` 类的逻辑。
- 但**不能整文件覆盖**。本项目的 `module/structurer.py` 和 `world_types.py` 已深度定制：
  - Pal 结构增加了反作弊字段（`talent_hp/melee/ranged/defense`、`soul_*`、`instance_id`、`equipped_skills` 等），并修正了参考项目把天赋误当战斗属性的问题。
  - Player 增加了 `platform_id` 提取（Steam/GDK 兼容）。
  - 使用的解析库不同：本项目依赖 `palworld-save-tools`（`from palworld_save_tools.gvas import GvasFile` 等），参考项目已迁移到更新的 `palsav` / `palsav-flex` / `palooz`（Oodle `PlM1` 解压，适配 Palworld 1.0）。两套 API 在 `ItemContainerSaveData` 的 RawData 访问路径上已有差异。
  - 因此正确做法是：**把参考项目的 `BaseCamp` 类和 `structure_base_camp()` 按本项目的 `palworld-save-tools` 数据形状移植进 `module/world_types.py` 和 `module/structurer.py`**，并在 Go 端 `database.Guild` 增加 `BaseCamp []BaseCamp` 字段。
- Go 端 `service/guild.go` 的 `PutGuilds/ListGuilds/GetGuild` 都是整段 JSON 序列化/反序列化，新增字段向后兼容，无需改 service 逻辑。
- 参考项目的 Go API 层（`api/guild.go`）逻辑与本项目几乎一致，无额外可借鉴处。

### 3.5 存档解析目前做到什么程度

- 玩家：昵称、等级、经验、HP/护盾、状态点、饱食度、帕鲁队伍（含天赋/灵魂/技能等反作弊字段）、6 类物品容器。
- 公会：名称、据等级、会长 UID、成员（含 last_online）、`base_ids`。
- 在线：实时坐标、ping、ip、等级（来自官方 REST，非存档）。
- **未做：据点（BaseCamp）坐标/范围、地图物件、采集物、Boss/塔位置、传送点位置（这些属于静态/世界数据，不在玩家存档里）。**

---

## 4. 地图瓦片现状

### 4.1 瓦片目录（`web/public/map/tiles/`）

路径格式 `tiles/{z}/{x}/{y}.webp`，全部为 256×256 像素 RGB WebP。

| 缩放 z | x 目录数 | y 文件数/列 | 总瓦片数 | 标准金字塔应有 |
|---|---|---|---|---|
| 0 | 1 | 1 | 1 | 1 (1×1) |
| 1 | 2 | 2 | 4 | 4 (2×2) |
| 2 | 4 | 4 | 16 | 16 (4×4) |
| 3 | 8 | 8 | 64 | 64 (8×8) |
| 4 | 8 | 8 | **64** | 256 (16×16) |

**注意 z=4 不完整**：x、y 都只有 0–7（即只覆盖 1/4 象限），标准 z=4 应有 16×16=256 张。瓦片金字塔在 z=4 处断裂，前端 `MAX_ZOOM=4` 时放大到 z=4 在地图右上区域会出现空白瓦片。这是重写时需要补齐或调整 maxZoom 的点。

格式：WebP（参考项目用的是从 palworld.gg 下载的 PNG）。本项目瓦片来源未在仓库内找到生成脚本（`_palworld-server-tool/map_down.py` 下载的是 PNG，且只到 z=6），推测 WebP 是经过离线转换+裁剪的。

### 4.2 treemap.webp

- 路径：`web/public/map/treemap.webp`
- 尺寸：**4096×4096 像素 RGB WebP**，文件大小约 1.18 MB。
- 在 `PlayerMap.vue` 中通过 `L.imageOverlay('/map/treemap.webp', [[0, 0], [86, 85]], { opacity: 0.9 })` 叠加在 Leaflet Simple CRS 地图左上角，作为一张半透明小地图/地形叠加层（bounds 86×85，约占 zoom=0 时 256×256 画布的 1/3）。
- 参考项目 `_palworld-server-tool/` 内**没有** treemap.webp，来源不是参考项目。从命名和尺寸看，应是一张 Palworld 全岛卫星/地形总图（可能是游戏内地图截图或社区地图），具体出处仓库内无记录。

### 4.3 后端静态路由（`main.go`）

```go
router.StaticFS("/assets", http.Dir(filepath.Join(webDir, "assets")))
router.StaticFS("/data",   http.Dir(filepath.Join(webDir, "data")))
router.StaticFS("/icons",  http.Dir(filepath.Join(webDir, "icons")))
router.StaticFS("/map",    http.Dir(filepath.Join(webDir, "map")))
```

`webDir` 自动探测 `web/dist`（生产构建）或 `web`（开发）。`/map/tiles/...` 和 `/map/treemap.webp` 即由 `/map` 静态服务提供。NoRoute 对非 `/api/` 路径回退到 `index.html`（SPA）。

---

## 5. 前端地图相关

### 5.1 `web/src/views/PlayerMap.vue`（当前 166 行）

现状要点：

- 只显示**在线玩家**：`onMounted` 调 `api.getOnline()`，`setInterval(loadOnline, 10000)` 每 10 秒刷新。
- 地图库：Leaflet，`L.CRS.Simple`（平面坐标系，非地理经纬度）。
- 常量：
  ```ts
  const TILE_SIZE = 256
  const MAX_ZOOM = 4
  const WORLD_HALF = 1400000
  ```
- 坐标变换 `gameToLatLng(x, y)`：把游戏世界坐标（范围假设 ±1,400,000）归一化到 [0,256]：
  ```ts
  const lat = ((-y + WORLD_HALF) / (WORLD_HALF * 2)) * TILE_SIZE
  const lng = ((x + WORLD_HALF) / (WORLD_HALF * 2)) * TILE_SIZE
  ```
- 瓦片层：`L.tileLayer('/map/tiles/{z}/{x}/{y}.webp', { tileSize:256, minZoom:0, maxZoom:4, noWrap:true, bounds })`。
- 叠加层：`L.imageOverlay('/map/treemap.webp', [[0,0],[86,85]], {opacity:0.9})`。
- 标记：每个在线玩家一个彩色圆点 `L.divIcon`，popup 显示 nickname / level / 坐标 / ping。无聚类、无公会/据点/离线玩家/Boss/传送点。
- 初始 `map.fitBounds([[0,0],[256,256]])` 后 `setZoom(1)`。

**坐标变换隐患**：本项目用对称的 `WORLD_HALF=1,400,000`，而参考项目 `MapView.vue` 用的是真实景观边界：

```js
// _palworld-server-tool/web/src/views/PcHome/component/MapView.vue
const LAND_SCAPE = [349400, 724400, -1099400, -724400]; // [maxX, maxY, minX, minY]
// x -> [-256, 0]; y -> [0, 256]
```

两者变换范围不一致。Palworld 实际世界边界约为 x∈[-1,099,400, 349,400]、y∈[-724,400, 724,400]（非对称）。本项目对称 1.4M 的换法会把岛屿整体推偏。重写时建议对齐参考项目的 `LAND_SCAPE` 边界。

### 5.2 路由与菜单

- 路由（`web/src/router/index.ts`）：
  ```ts
  { path: 'playermap', name: 'playermap', component: () => import('@/views/PlayerMap.vue') }
  ```
  完整路径 `/playermap`。
- 菜单（`web/src/views/Layout.vue` 第 85 行）：
  ```ts
  { label: '玩家地图', key: '/playermap' }
  ```
  入口名称叫「玩家地图」。

### 5.3 Leaflet 依赖（`web/package.json`）

已存在：

```json
"leaflet": "^1.9.4",
"@types/leaflet": "^1.9.22"
```

无需新增。Vue 版本 3.4，vue-router 4.3，naive-ui 2.38，构建工具 Vite 5。

### 5.4 可直接复用的前端 API 方法

`api.getOnline()`、`api.getPlayers()`、`api.getGuilds()`、`api.getGuild(uid)` 均可用于地图重写。当前 `PlayerMap.vue` 只用了 `getOnline()`。重写后若要展示离线玩家最后位置、公会据点，需要调用 `getPlayers()` 和 `getGuilds()`（待后端补 `base_camp` 字段后）。

---

## 6. 静态数据

### 6.1 `web/public/data/`

| 文件 | 大小 | 结构 | 用途 |
|------|------|------|------|
| `pal_list.json` | 32 KB | `[{id, name, icon}]` | 帕鲁名录 |
| `item_list.json` | 109 KB | `[{id, name, icon}]` | 物品名录 |
| `tech_list.json` | 55 KB | `[{id, name, icon}]` | 科技名录 |

`web/public/icons/` 下有 `items/`、`pals/`、`tech/` 三个 PNG 图标子目录。

**没有任何地图坐标类数据**：无 Boss 塔坐标、无传送点坐标、无据点/聚落坐标。

### 6.2 参考项目可复用的地图静态数据

参考项目 `_palworld-server-tool/web/src/assets/map/` 下有现成资源：

- `points.json`：含两类坐标（游戏世界坐标，与玩家/据点坐标同系）：
  - `boss_tower`：9 个 `[x, y]` 坐标（如 `[-266563.2, 174506.3]`）。
  - `fast_travel`：约 120+ 个 `[x, y]` 传送点坐标。
- 标记图标（WebP）：
  - `base.webp` / `base_offline.webp` / `base_online.webp`：据点（在线/离线）。
  - `boss_tower.webp`：Boss 塔。
  - `fast_travel.webp`：传送点。
  - `player.webp`：玩家。

这些坐标是静态的（游戏版本相关，不随存档变化），可直接拷贝到本项目 `web/public/data/`（或 `web/public/map/`）供新地图使用。注意参考项目用 `LAND_SCAPE` 变换，本项目若采用相同变换即可无缝对齐。

参考项目 `_palworld-server-tool/web/src/views/PcHome/component/MapView.vue` 还实现了：据点标记（按公会分组）、在线/全部玩家切换、Boss 塔/传送点图层开关、玩家搜索定位、距离换算等，可作为前端重写的功能参考。

---

## 7. 已有能力清单

1. **在线玩家实时定位**：官方 REST `/v1/api/players` 提供 `location_x/location_y/level/ping/nickname/player_uid`，后端 `tool.ShowPlayers()` 与 `GET /api/online_player` 已通，前端 `api.getOnline()` 已封装。默认 60s 轮询入库、前端 10s 刷新。
2. **全量玩家数据 API**：`GET /api/player` 返回存档解析出的全部玩家（精简结构），`GET /api/player/:uid` 返回含帕鲁/物品的详情。结构上嵌入了 `OnlinePlayer`，能保留最后一次在线坐标。
3. **公会列表 API**：`GET /api/guild` / `GET /api/guild/:uid` 已存在，返回公会名、据等级、会长、成员、`base_ids`。
4. **存档解析流水线已通**：Python sav_cli（`module/`，基于 palworld-save-tools）由 Go 端 `tool.Decode()` 调度，PyInstaller 打包，定时（默认 120s）解析 Level.sav + Players/*.sav 并 PUT 回后端，写入 bbolt。
5. **地图瓦片已具备 0–3 完整、4 部分**：`web/public/map/tiles/{z}/{x}/{y}.webp`（256×256 WebP），后端 `/map` 静态路由已注册。
6. **treemap.webp 地形叠加层**：4096×4096 WebP，已在地图上叠加。
7. **前端 Leaflet 已集成**：`leaflet ^1.9.4` + `@types/leaflet`，`PlayerMap.vue` 已实现 Simple CRS、瓦片层、在线玩家标记、popup、自动刷新。
8. **路由/菜单入口已存在**：`/playermap`，菜单名「玩家地图」。
9. **静态游戏数据**：帕鲁/物品/科技三个 JSON 名录及图标。
10. **参考项目可直接借鉴/移植**：`_palworld-server-tool/sav_cli` 有完整的 `BaseCampSaveData` 解析与 `base_camp` 坐标输出；`web/src/assets/map/points.json` 有 Boss 塔和传送点坐标；`MapView.vue` 有完整地图交互参考；地图瓦片可从 palworld.gg 下载到 z=6。

---

## 8. 缺失能力清单

1. **据点（BaseCamp）坐标完全未解析**：
   - `module/structurer.py` 未读 `BaseCampSaveData`；`module/world_types.py` 无 `BaseCamp` 类，`Guild` 无 `base_camp` 字段。
   - Go 端 `database.Guild` 无 `BaseCamp` 字段（只有 `base_ids`）。
   - 需移植参考项目的 `structure_base_camp()` + `BaseCamp` 类到本项目 sav_cli，并在 Go model 增加 `BaseCamp []BaseCamp{Id,Area,LocationX,LocationY}`。
2. **离线玩家无存档坐标**：存档解析未提取玩家位置（CharacterSaveParameter 中也未取坐标），离线玩家坐标仅依赖在线轮询快照。若要展示离线玩家位置，需另行研究存档中是否有角色坐标可解析（参考项目同样只靠在线坐标 + 据点）。
3. **地图瓦片金字塔不完整**：z=4 只有 8×8=64 张（应为 16×16），且最高只到 z=4。参考项目到 z=6。重写需补齐瓦片或降低/调整 maxZoom。
4. **无 Boss 塔 / 传送点静态坐标数据**：`web/public/data/` 下没有。需从参考项目 `points.json` 引入（9 Boss 塔 + 约 120 传送点）及对应标记图标。
5. **前端地图功能单薄**：`PlayerMap.vue` 仅在线玩家圆点，无据点图层、无公会着色/分组、无离线玩家、无 Boss/传送点图层开关、无搜索定位、无标记聚类、无坐标换算工具。
6. **坐标变换与真实世界边界不符**：当前用对称 `WORLD_HALF=1400000`，与 Palworld 实际非对称边界（参考项目 `LAND_SCAPE=[349400,724400,-1099400,-724400]`）不一致，会导致标记与瓦片错位。重写需统一坐标系。
7. **sav_cli 解析库版本偏旧**：本项目用 `palworld-save-tools`，参考项目已迁移到 `palsav`/`palooz`（适配 Palworld 1.0 Oodle 解压）。移植据点解析时需按本项目所用库的 RawData 数据形状调整，并验证对当前游戏版本存档的兼容性。
8. **treemap.webp 来源/授权不明**：仓库内无生成脚本或出处说明，重写若要继续使用需确认来源与授权，或改用 palworld.gg 标准瓦片。
