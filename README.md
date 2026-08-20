# PalAdmin

幻兽帕鲁（Palworld）Windows 服务器管理与反作弊面板。

- 🪟 **Windows 原生**：面板直接运行在 Windows 上，调用本机 SteamCMD 下载/启动 `PalServer.exe`，无需 Linux、Proton 或 Wine
- 🛡️ **PalDefender 反作弊**：实时拦截属性修改、非法物品、违禁科技等作弊，支持踢出/封禁/IP 封禁；面板内一键安装
- 🎮 **SteamCMD 管理**：面板内一键安装 SteamCMD、下载/更新服务端，启动、停止、重启，无需手写命令
- 🗺️ **交互地图**：Leaflet 世界地图（主世界 / 天坠之地），实时显示在线玩家、据点、Boss 塔、快速旅行点
- 👥 **玩家管理**：在线/存档玩家列表，查看背包物品（WebP 图标）、帕鲁详情，踢/封/解封
- 🏛️ **公会**：公会信息与成员列表
- 💾 **存档备份与回档**：定时备份，一键回滚
- ⚙️ **可视化配置**：`PalWorldSettings.ini` 全部参数按分类编辑，字段含中文说明和取值范围
- 🌗 **暗色主题**、响应式、手机端适配
- 🔄 **在线自更新**：经国内镜像测速下载，一键升级

## 架构

```
浏览器（Vue3 + Naive UI）
        │  http://localhost:8190 或 http://服务器IP:8190
        ▼
PalAdmin（paladmin.exe，Windows 进程）
        ├── SteamCMD  →  下载/更新 Windows 版游戏服（App 2394010）
        ├── 直接启动  →  PalServer-Win64-Shipping-Cmd.exe
        ├── PalDefender DLL 注入游戏进程
        └── REST API  →  连接游戏服 :8212 同步玩家/执行操作
```

**关键路径（默认，用户可在面板修改）：**
- SteamCMD 目录：如 `C:\steamcmd`
- 游戏安装目录：如 `C:\PalServer`
- 游戏可执行文件：`<游戏目录>\Pal\Binaries\Win64\PalServer-Win64-Shipping-Cmd.exe`
- 配置文件：`<游戏目录>\Pal\Saved\Config\WindowsServer\PalWorldSettings.ini`
- 存档：`<游戏目录>\Pal\Saved\SaveGames\`
- PalDefender DLL：`<游戏目录>\Pal\Binaries\Win64\`（d3d9.dll / PalDefender.dll）

## 部署

### 前置条件

- Windows 10/11 或 Windows Server 2019+（amd64）
- 开放端口：面板 `8190/tcp`、游戏 `8211/udp`、REST `8212/tcp`（REST 建议仅内网）
- 首次运行 SteamCMD 下载游戏服需要网络（建议提前装好 Visual C++ 运行库）

### 安装

1. 从 [GitHub Release](https://github.com/QYC-qyc/palworld-tool/releases) 下载 `paladmin_windows_amd64.zip"
2. 解压到任意目录，如 `C:\PalAdmin`
3. 双击 `paladmin.exe` 启动（或在命令行运行查看日志）
4. 浏览器访问 `http://localhost:8190`，首次设置面板登录密码

> 面板默认监听 `0.0.0.0:8190`，同局域网其他机器可通过 `http://<服务器IP>:8190` 访问。生产环境建议用防火墙限制 8190 端口来源。

### 开机自启动（可选）

用任务计划程序创建基本任务：触发器选"计算机启动时"，操作选"启动程序"，浏览选择 `paladmin.exe`，勾选"使用最高权限运行"。

## 使用流程

1. **设置面板密码**（首次访问）
2. **「游戏服」页**填写路径：
   - SteamCMD 目录：如 `C:\steamcmd`（不存在时点「安装」按钮自动下载安装）
   - 游戏安装目录：如 `C:\PalServer`
   - 保存配置
3. **安装 SteamCMD**（若未安装）：点 SteamCMD 目录旁的「安装」，自动下载 steamcmd.zip、解压并完成首次自更新
4. **「游戏服」页 → 安装/更新游戏服**：SteamCMD 下载 Windows 版帕鲁服务端，日志区实时显示进度
5. **「PalDefender」页 → 安装 PalDefender**（可选）：下载 DLL 到 Win64 目录；在「设置」页配置反作弊开关（踢出/封禁/IP封禁），保存时写入 PalDefender/Config.json 并热重载
6. **「游戏服」页 → 启动**（直接启动 Windows 版 exe，自动加载 PalDefender）
7. **「游戏配置」页**开启 REST API：`RESTAPIEnabled=true`、`RESTAPIPort=8212`、设置 `AdminPassword`，保存并重启
8. 完成后仪表盘/地图/玩家页即可同步数据

> 启动前会检查游戏 exe 是否存在；PalDefender DLL 缺失仅警告不阻止启动。

### 防火墙

```powershell
New-NetFirewallRule -DisplayName "PalAdmin" -Direction Inbound -LocalPort 8190 -Protocol TCP -Action Allow
New-NetFirewallRule -DisplayName "Palworld" -Direction Inbound -LocalPort 8211 -Protocol UDP -Action Allow
```

云服务器还需在安全组放行相同端口。

## 功能说明

| 功能 | 说明 |
|---|---|
| 仪表盘 | 服务器名/版本/在线/FPS 统计、控制台（广播/同步/关服） |
| 游戏服 | SteamCMD 与安装目录配置、一键安装 SteamCMD、安装/更新 Windows 版、启停重启、实时日志 |
| PalDefender | 安装/卸载 PalDefender DLL、生成 Token；玩家管理、封禁列表、广播警报、公会据点、配置热重载 |
| 游戏配置 | 可视化编辑 PalWorldSettings.ini，参数分类带说明 |
| 玩家 | 在线/存档玩家 Tab、搜索、详情（基本信息/帕鲁/物品）、踢/封/解封/IP封禁（走 PalDefender） |
| 玩家地图 | Leaflet 地图，玩家/据点/Boss塔/快速旅行点，主世界/天坠之地切换 |
| 公会 | 公会与成员列表 |
| 白名单 | 白名单管理 |
| 备份 | 存档备份列表、一键回档；定时备份 |
| 审计 | 操作日志 |
| 设置 | 面板密码、REST/PalDefender 连接、反作弊开关、存档路径、面板在线更新 |

## 在线更新

面板「设置」页可检查并一键升级：后端会对多个 GitHub 镜像测速，选最快的下载 `paladmin_windows_amd64.zip`，通过临时 .bat 脚本自动替换 exe 并重启。

## 开发

- **后端**：Go，入口 `main.go`；路由 `api/router.go`；游戏服管理 `internal/gamesrv/`；PalDefender 代理 `api/paldefender*.go`；存档解析 `internal/tool/`
- **前端**：Vue 3 + TypeScript + Vite + Naive UI，代码在 `web/`
- **详细约定**：见 [docs/开发约定.md](docs/开发约定.md)
- 构建：
  ```bash
  cd web && npm install && npm run build && cd ..
  GOOS=windows GOARCH=amd64 go build -o paladmin.exe .
  ```
- GitHub Actions 在打 tag 时自动构建 `paladmin_windows_amd64.zip` 并发版

## 致谢

- [PalDefender](https://github.com/Ultimeit/PalDefender) — 反作弊
- [palworld-save-tools](https://github.com/cheahjs/palworld-save-tools) — 存档格式解析
- [LootLab](https://lootlab.cn/palworld) — 配置参数范围、图标、地图数据

## 许可证

MIT
