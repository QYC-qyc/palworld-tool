# Docker 部署兼容性审计报告

> 审计日期：2026-08-21
> 审计范围：v3.0.0 Docker 双容器架构（paladmin + gameserver），对照源码、Dockerfile、compose、CI 逐一核实
> 修复日期：2026-08-21（v3.0.1），下文 P0/P1 问题均已修复并通过 `go build`/`go vet`/`npm run build`/`docker compose config` 验证。
>
> 修复总览：
> - 面板镜像补齐前端资源与 sav_cli；gameserver entrypoint 修正 steamcmd 路径并启用官方 REST API
> - 面板入口脚本把数据库/备份/日志重定向到 /data，并映射 REST/PalDefender/进程控制等环境变量
> - 新增 `internal/gamesrv/gamefs.go` 跨容器文件访问层（本地走 os、Docker 走 `docker exec`/`docker cp`），
>   游戏配置、PalDefender 安装/Token、备份、回档全部改走该层
> - 自动备份改用 Manager 判断运行状态；新增备份保留清理；容器内禁用面板自更新

## 〇、总体结论

架构层面存在断裂：paladmin 容器通过挂载的 docker.sock **只能控制 gameserver 容器的生命周期**（start/stop/restart/logs 这 4 个功能是好的），但所有**文件级操作**——读写游戏配置 ini、安装 PalDefender DLL、备份/回档存档、生成 Token、写反作弊配置——全部在 paladmin 容器本地文件系统执行，而这些文件实际在 gameserver 容器内，两容器没有共享卷，所以全部写错位置。

叠加镜像打包缺失和环境变量不被读取，导致：

- 拉取 ghcr 镜像 `docker compose up` 后，访问 :8190 是**空白页**（无前端资源）
- gameserver 容器因 steamcmd 路径错误，**首启自动安装游戏服失败**，反复重启
- 即便手动装好，游戏服也**没开 REST API**，面板连不上
- 面板数据库写在容器层，重建容器**全部丢失**

---

## 一、阻断级问题（P0，不修复则完全不可用）

### P0-1　paladmin 镜像没有前端资源，页面空白
- `docker/paladmin/Dockerfile:30-31` 运行阶段只 COPY 了二进制和 entrypoint.sh。
- 主程序无 `go:embed`（全仓 grep 仅 `_palworld-server-tool` 有），`main.go:73-98` 运行时找 `web.dat` 或 `/app/web/dist`，镜像里都没有 → 打印"未找到前端资源，仅提供 API"。
- CI `release.yml:129` 正是用这个 Dockerfile 构建并推送 `ghcr.io/qyc-qyc/paladmin`，**已发布的 v3.0.0 镜像受影响**。
- 修复：运行阶段加 `COPY --from=web-builder /src/web/dist /app/web/dist`。

### P0-2　gameserver 的 steamcmd 路径错误，安装/更新/首启必失败
- `docker/gameserver/Dockerfile:27-32` 把 steamcmd 装在 `/opt/steamcmd/steamcmd.sh`，软链到 `/usr/local/bin/steamcmd`。
- `docker/gameserver/entrypoint.sh:9` 却写死 `STEAMCMD="/usr/games/steamcmd"`——该路径不存在。
- 脚本 `set -e`，首启自动安装（entrypoint.sh:28-31）和面板"更新游戏服"（exec entrypoint.sh update）都会 "No such file or directory" 退出。
- 修复：改为 `STEAMCMD="steamcmd"`（/usr/local/bin 在 PATH）或 `/opt/steamcmd/steamcmd.sh`。

### P0-3　gameserver 启动未启用官方 REST API，8212 无监听
- `docker/gameserver/entrypoint.sh:46-51` 启动参数只有 `-port -publiclobby -useperfthreads -NoAsyncLoadingThread -UseMultithreadForDS`，**没有 `-RESTAPI`**。
- `PalWorldSettings.ini` 中 `RESTAPIEnabled` 默认 False（`internal/palconfig/schema.go:123`）。
- compose 映射了 8212 但没有进程监听，面板所有 REST 功能（在线玩家、广播、关服、信息）都连不上。
- 修复：启动参数加 `-RESTAPI -RESTPort=8212 -RESTPassword=<密码>`，或在初始化时写 ini 开启。

### P0-4　GAMESERVER_URL 环境变量 Go 代码完全不读
- compose（`docker-compose.yml:51`、prod:46）设了 `GAMESERVER_URL=http://gameserver:8212`，但全仓 Go 代码 **grep 零匹配**。
- REST 地址实际取 `viper.GetString("rest.address")`（`internal/tool/rest_api.go:21`），默认空串（`service/settings.go:33`），镜像又没有 config.yaml → 地址为空，请求必败。
- 修复：entrypoint.sh 加 `export REST__ADDRESS="${GAMESERVER_URL}"`（viper 用 `__` 替换 `.`，见 `internal/config/config.go` 的 AutomaticEnv）。

### P0-5　/data 持久卷不被使用，数据库/备份重建即丢
- `docker/paladmin/entrypoint.sh:5-6` 设了 `PALADIN_DATA_DIR=/data`，但 Go 代码 grep `PALADIN_DATA_DIR` **零匹配**。
- `main.go:47-50` 数据库默认 `./pst.db`，WORKDIR 是 /app → 实际 `/app/pst.db`（容器读写层）。
- `getDataDir()`（main.go:173-183）Linux 下返回 exe 目录 /app，不读该环境变量。
- compose 挂了 `./data/paladmin:/data` 但程序从不写 /data → 账号、设置、玩家/公会/白名单、备份元数据全部在容器层，`up --force-recreate` 或升级即丢。文档 `docs/开发规范.md:94` 称"/data/pst.db"与代码不符。
- 修复：entrypoint.sh 加 `export STORAGE__PATH=/data/pst.db`；备份目录指向 /data/backups；日志指向 /data/logs；或让代码识别 `PALADIN_DATA_DIR`。

### P0-6　两容器无共享卷，所有游戏文件操作写错容器
- paladmin 挂 `./data/paladmin:/data`，gameserver 挂 `./data/gameserver:/home/steam/palserver`，互不共享。
- 以下功能全部用 `os.ReadFile/WriteFile/Stat` 操作 paladmin 容器本地路径，gameserver 根本看不到：
  - 游戏配置读写（`api/gamesettings.go:57,102,140`）
  - 备份（`internal/tool/save.go`）、回档（`service/restore.go`）
  - PalDefender 安装/卸载（`api/paldefender.go:106,145,211`）
  - Token 生成（`api/paldefender_api.go:62,99`）、反作弊配置（同文件:418）
- 当 InstallDir 为空时，iniPath 还会退化成 Windows 路径 `C:\PalServer\...`（gamesettings.go:16），在 Linux 容器里造出名字带冒号的相对目录。
- 修复方向（二选一）：
  - **方案 A（简单）**：paladmin 也挂 `./data/gameserver:/home/steam/palserver`，并设 `GAMESRV__INSTALL_DIR=/home/steam/palserver`、`SAVE__PATH=/home/steam/palserver/Pal/Saved`。需处理 steam 用户(容器内 uid 1000)对宿主文件的权限。
  - **方案 B（更安全，推荐）**：扩展 `internal/gamesrv/docker.go`，增加 readFile/writeFile/statPath/extractZip，用 `docker exec`/`docker cp` 操作 gameserver 容器内文件；上述功能全部改走该层。

### P0-7　回档"假成功"：不停服、写错误位置、返回成功
- `service/restore.go:24` 用 `system.NewProcessCtl()`，读 `process.mode`，默认 `noop`（`internal/system/process.go:20-30`），Stop/Start 是空操作、IsRunning 恒 true。
- compose 只设了 `GAMESERVER_CONTAINER`（gamesrv 包的开关），**没设** `PROCESS__MODE=docker`（system 包的开关），两套 docker 开关未统一。
- 结果：回档既不停 gameserver，又把存档解压到 paladmin 容器的无效路径，却返回成功，极具误导性。
- 修复：让 `system.NewProcessCtl` 在检测到 `GAMESERVER_CONTAINER` 时自动用 docker 驱动并复用 gamesrv.Manager 的启停；存档操作配合 P0-6 修。

### P0-8　paladmin 镜像缺 sav_cli，存档解析/地图/同步全失效
- `internal/tool/save.go:25-41` 找工作目录下的 sav_cli，找不到就报错。`docker/paladmin/Dockerfile` 没有该构建阶段（根目录 `Dockerfile:20-25,37,45` 才有）。
- 影响：`/api/sync`、定时存档同步、玩家/公会离线数据全部失败。
- 附带问题：CI 的 sav_cli 在 ubuntu runner 上只编一次（amd64 ELF），却被打进 arm64 tar 包（`release.yml:40-59` 在架构分叉前构建），arm64 存档解析也坏（开发规范第 8 条坑只在 Windows 修了）。
- 修复：paladmin Dockerfile 增加 sav_cli 阶段（按目标架构），设 `SAVE__DECODE_PATH=/app/sav_cli`；arm64 单独编。

### P0-9　自动备份判定用 pgrep，跨容器恒 false
- `internal/task/task.go:75-89` `isGameRunning()` 在面板容器里 `pgrep -f PalServer-Win64-Shipping-Cmd.exe`，进程在 gameserver 容器，pid namespace 隔离 → 恒返回 false。
- `backupTask`（task.go:61-64）每次都跳过，自动备份永不执行。
- 修复：改调 `gamesrv.Manager.GetStatus()` 或 `dockerCtl.isRunning()`。

---

## 二、功能失效（P1，核心功能不可用）

| # | 问题 | 证据 |
|---|------|------|
| P1-1 | PalDefender REST 默认 127.0.0.1:17993，面板连到自身而非 gameserver；且 PD 需绑 0.0.0.0 | `internal/paldefender/client.go:42-48`、`service/settings.go:41-42` |
| P1-2 | PalDefender DLL 镜像未预装，面板又写错容器，反作弊实际加载不到（entrypoint.sh:38-43 会告警无 DLL） | `docker/gameserver/Dockerfile`、`api/paldefender.go` |
| P1-3 | Setup 向导默认 `http://palworld:8212`，容器名/服务名是 `gameserver`，Docker DNS 不解析 `palworld` | `web/src/views/Setup.vue:76`、`api/gamesettings.go:186` |
| P1-4 | "安装 SteamCMD"按钮无 docker 分支，在 paladmin 容器内下载 steamcmd，污染面板且对游戏服无用（游戏服镜像已预装） | `internal/gamesrv/manager.go:172-192` |
| P1-5 | 面板"在线更新"在容器内无效：无 systemd、/app 是镜像层重启还原、PID 1 退出连带杀死替换脚本 | `internal/updater/replace_unix.go:35-42`、`api/updater.go:72` |
| P1-6 | 备份文件写 /app/backups（容器层），且 backup.keep_count/keep_days 从未被代码读取，无轮转，会撑爆磁盘 | `internal/tool/save.go:107`、config.example.yaml:35-37 |
| P1-7 | `save.path` 默认 `/home/steam/Pal/Saved` 少了 palserver 段，真实路径是 `/home/steam/palserver/Pal/Saved` | `config.example.yaml:26`、`config.yaml:30` |
| P1-8 | "额外启动参数"`m.cfg.ExtraArgs` 在 docker 模式被静默忽略（参数由 entrypoint 硬编码） | `internal/gamesrv/manager.go:531-540` |
| P1-9 | Start 的 docker 分支无启动后存活探测，容器 entrypoint 崩了也报启动成功 | `internal/gamesrv/manager.go` Start docker 分支 |

---

## 三、体验/健壮性问题（P2）

- `checkProtonReady` 提示"一键安装 Proton"，但后端 `installProton` 函数并不存在（空头支票）。
- 前端平台标识固定显示 Windows（`GameServer.vue:42`），未体现 Proton/Docker。
- 前端路径配置表单（SteamCMD 目录、游戏安装目录，placeholder 是 `C:\...`）在 Docker 模式无意义，应隐藏或只读。
- 两个 compose 文件（`docker-compose.yml` 与 `docker-compose.prod.yml`）高度重复，唯一差异是 build vs image，易不同步。
- 备份无下载/上传接口（router.go 无 FormFile），用户无法在 Docker 下导入导出存档来规避上述缺陷。
- `api/gamesettings.go:207` 在判定 host 为本机时会把 rest.address 改写成 127.0.0.1，逻辑脆弱。

---

## 四、正常的部分（无需改动）

- gamesrv.Manager 的 **Start/Stop/Restart/Install/GetStatus/Logs** 的 docker 分支结构正确，通过 docker.sock 调用 docker CLI，方向无误（Install 受 P0-2 影响是 entrypoint 的锅）。
- **前端所有 API 请求走同源相对路径** `/api/...`，由后端代理；无浏览器直连 `gameserver:8212`、无硬编码 ws://，架构正确（B5）。
- 静态资源（图标/瓦片/数据）在 web/dist 内构建齐全，只要 COPY 进镜像即可正常服务。
- 定时任务中在线玩家同步、踢非白名单走 REST，REST 配好后可用。

---

## 五、修复优先级与建议顺序

**第一阶段（让 Docker 能跑起来，最小可用）：**
1. P0-2 entrypoint.sh steamcmd 路径（1 行）
2. P0-1 paladmin Dockerfile COPY web/dist（1 行）
3. P0-3 entrypoint 启动参数加 -RESTAPI
4. P0-4 entrypoint 导出 REST__ADDRESS
5. P0-5 entrypoint 导出 STORAGE__PATH，备份/日志归 /data
6. P0-8 paladmin Dockerfile 加 sav_cli
7. P1-3 Setup 默认地址改 gameserver
8. P1-1 PalDefender 默认 host 改 gameserver

**第二阶段（文件操作跨容器）：**
9. P0-6 选定方案 A（共享卷，快）或 B（docker cp/exec，干净）并实施
10. P0-7 统一 process.mode 与 GAMESERVER_CONTAINER 两套开关
11. P0-9 isGameRunning 改走 manager
12. P1-4/P1-5 Docker 模式下前端隐藏安装 SteamCMD / 自更新按钮
13. P1-6 备份轮转 + 迁移到 /data

**第三阶段（打磨）：** P1-8/P1-9/P2 各项。

建议完成第一阶段后立即在干净环境 `docker compose down -v && docker compose build --no-cache && docker compose up` 实测：能打开页面、能装上游戏服、能看到在线玩家、重启容器数据不丢。再进入第二阶段。
