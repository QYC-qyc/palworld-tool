# PalAdmin

幻兽帕鲁（Palworld）服务器管理与反作弊面板。**Docker 一键部署**：一个命令同时启动面板和游戏服，在网页里完成安装、启停、配置、玩家管理、反作弊、备份回档。

- 🐳 **Docker 一键部署**：`docker compose up -d` 启动面板 + 游戏服，数据自动持久化
- 🛡️ **PalDefender 反作弊**：实时拦截属性修改、非法物品、违禁科技，支持踢出 / 封禁 / IP 封禁
- 🎮 **游戏服管理**：网页内安装 / 更新 Windows 版服务端、启停重启、实时日志
- 🗺️ **交互地图**：世界地图（主世界 / 天坠之地），实时显示在线玩家、据点、Boss 塔、快速旅行点
- 👥 **玩家管理**：在线 / 存档玩家列表，查看背包物品与帕鲁详情，踢 / 封 / 解封
- 🏛️ **公会**：公会信息与成员列表
- 💾 **备份与回档**：自动定时备份，一键回滚（自动停服 → 备份 → 恢复 → 启服）
- ⚙️ **可视化配置**：`PalWorldSettings.ini` 全参数分类编辑，带中文说明与取值范围
- 🌗 **暗色主题**、响应式、手机端适配

---

## 一、安装部署

### 1. 准备服务器

- **系统**：Linux x86_64（推荐 Ubuntu 22.04 / Debian 12）。ARM64 可运行但需 box64 转译，兼容性不保证。
- **配置**：至少 2 核 CPU、4GB 内存、10GB 磁盘（游戏服约占 5GB）。
- **已安装 Docker 与 Compose 插件**，未安装可执行：
  ```bash
  curl -fsSL https://get.docker.com | sh
  ```
- **开放端口**（云服务器安全组也要放行）：
  | 端口 | 协议 | 用途 |
  |---|---|---|
  | 8190 | TCP | 面板网页 |
  | 8211 | UDP | 游戏端口（玩家连接） |
  | 8212 | TCP | 游戏 REST API（建议仅内网，不必对公网开放） |

### 2. 一条命令启动

在你想存放数据的目录下（例如 `~/paladmin`）执行：

```bash
mkdir -p ~/paladmin && cd ~/paladmin
curl -fsSL -o docker-compose.yml \
  https://gitee.com/qyc-qyc/palworld-tool/raw/main/docker-compose.prod.yml
docker compose up -d
```

首次启动会自动拉取两个镜像（约 1GB，国内网络可能需要几分钟）：

- `paladmin`：管理面板
- `palworld-gameserver`：游戏服（内含 SteamCMD + GE-Proton + PalServer）

查看启动状态：

```bash
docker compose ps            # 两个容器应均为 Up
docker compose logs -f gameserver   # 看游戏服安装进度（首次会自动下载游戏）
```

### 3. 访问面板

浏览器打开 **`http://服务器IP:8190`**，首次访问会要求设置管理员密码，设置后用该密码登录。

---

## 二、首次使用向导

登录后按以下顺序操作即可开服：

1. **安装 / 更新游戏服**
   进入「游戏服」页，点击 **「安装 / 更新游戏服」**。面板会在游戏服容器内通过 SteamCMD 下载 Windows 版服务端（约 3GB，请耐心等待，可在弹窗或日志中看进度）。

2. **安装 PalDefender 反作弊（可选但推荐）**
   进入「PalDefender」页，点击 **「安装 PalDefender」**，面板会把反作弊 DLL 下载到游戏服目录。
   安装后点击 **「生成 Token」**，复制生成的 Token 备用。

3. **启动游戏服**
   回到「游戏服」页点击 **「启动」**。状态变为「运行中」即成功。

4. **配置反作弊连接（安装了 PalDefender 才需要）**
   进入「设置」页，在 PalDefender 区域：
   - 主机填 `gameserver`（Docker 内部网络容器名，**不要填 127.0.0.1**）
   - 端口默认 `17993`
   - 粘贴第 2 步生成的 Token
   保存。之后封禁 / 广播 / 私聊等功能即生效。

5. **配置游戏参数**
   进入「游戏配置」页，按 8 个分类（服务器、世界、玩家、帕鲁、战斗、经济掉落、据点、公会）修改，保存后按需重启游戏服。

> 官方 REST API 已由容器启动参数自动启用（端口 8212，密码见下方配置），面板默认已连好，**无需**手动到游戏配置里开 `RESTAPIEnabled`。

---

## 三、常用配置

### 修改 REST 密码 / 面板端口等

在 `docker-compose.yml` 同目录创建 `.env` 文件可覆盖默认配置：

```bash
# REST API 密码（同时作为游戏服 REST 连接密码，默认 paladmin，建议修改）
REST_PASSWORD=你的强密码
```

> 改完 `.env` 执行 `docker compose up -d` 即可生效。
> 如需改面板对外端口，编辑 `docker-compose.yml` 里 `8190:8190` 左侧的宿主机端口即可。

### 数据存在哪

所有数据都在启动目录的 `./data` 下，**删除容器不会丢档**：

```
./data/
├── gameserver/   # 游戏安装、存档、配置、PalDefender DLL
└── paladmin/     # 面板数据库、备份 zip、日志
```

**备份整个服务器**：只需备份 `./data` 目录。建议定期打包下载。

---

## 四、功能说明

| 页面 | 能做什么 |
|---|---|
| 仪表盘 | 服务器名 / 版本 / 在线人数 / FPS，全服广播、手动同步、关服 |
| 游戏服 | 启停重启、安装 / 更新服务端、查看实时日志、查看运行模式 |
| PalDefender | 安装 / 卸载反作弊、生成 API Token、封禁列表、广播 / 警报 / 私聊、公会据点、配置热重载 |
| 游戏配置 | 可视化编辑全部 `PalWorldSettings.ini` 参数，难度一键预设 |
| 玩家 | 在线 / 存档玩家、搜索、查看背包与帕鲁、踢出 / 封禁 / 解封 / 封 IP |
| 玩家地图 | 世界地图上显示玩家、据点、Boss 塔、传送点，可切主世界 / 天坠之地 |
| 公会 | 公会列表与成员详情 |
| 白名单 | 白名单管理，可开启「非白名单自动踢出」 |
| 备份 | 备份列表、一键回档；后台按设定间隔自动备份并按数量 / 天数清理 |
| 审计 | 所有管理操作的日志 |
| 设置 | 面板密码、REST 连接、PalDefender 连接、反作弊处置开关、备份策略 |

---

## 五、日常运维

```bash
docker compose logs -f paladmin      # 面板日志
docker compose logs -f gameserver    # 游戏服日志（网页内也能看）
docker compose restart paladmin      # 重启面板
docker compose restart gameserver    # 重启游戏服
docker compose down                  # 停止全部（数据保留）
docker compose up -d                 # 启动
docker compose pull && docker compose up -d   # 更新到最新版镜像
```

**进入游戏服容器排查：**
```bash
docker compose exec gameserver bash
```

**手动在容器内更新游戏服：**
```bash
docker compose exec gameserver /home/steam/entrypoint.sh update
```

---

## 六、常见问题

**Q：玩家连不上服务器？**
检查：游戏服状态为「运行中」；云服务器安全组放行 8211/UDP；服务器防火墙放行该端口。

**Q：面板显示在线玩家但操作（踢人 / 广播）失败？**
多半是 PalDefender 未正确连接。确认已安装 DLL、已生成 Token、设置里主机是 `gameserver` 而非 `127.0.0.1`，且游戏服已重启加载过 DLL。

**Q：安装游戏服很慢或失败？**
首次需要从 Steam 下载约 3GB。若卡在 SteamCMD 自更新，通常是服务器到 Steam 的网络不通，可重试，或为 Docker 配置网络代理。

**Q：更新面板会丢数据吗？**
不会。数据库、备份、存档都在 `./data` 目录，镜像更新只替换程序。更新用 `docker compose pull && docker compose up -d`。

**Q：REST API 的 8212 端口要对公网开放吗？**
不需要。面板通过 Docker 内部网络访问它，不要对公网开放 8212。

**Q：忘了面板密码怎么办？**
目前没有找回功能。可停止容器后删除 `./data/paladmin/pst.db` 重新初始化（会清空面板设置，但不影响游戏存档），或直接 SSH 进数据库修改。

---

## 许可

MIT
