# 部署指南（Ubuntu 22.04）

## 方式一：Docker Compose（推荐）

前置：已安装 Docker，游戏服可通过容器名 `palworld` 访问。

```bash
# 1. 复制并修改环境变量
cp .env.example .env
vim .env   # 设置 WEB_PASSWORD、GAME_ADMIN_PASSWORD 等

# 2. 修改 config.yaml 中的 save.path（compose 已挂载为 /game/Saved）
#    进程控制已配置为 docker，回档时会自动停/启 palworld 容器

# 3. 构建并启动
docker compose up -d --build
docker compose logs -f paladmin
```

访问 `http://<服务器IP>:8080`，用 `WEB_PASSWORD` 登录。

### 游戏服也在 Docker
若游戏服与 paladmin 在同一 compose 网络，`rest.address` 用 `http://palworld:8212`，
`rcon.address` 用 `palworld:25575`，并在 compose 中把游戏服务加入同一网络。

### PalDefender 集成（Wine/Proton 游戏服）
在游戏容器安装 PalDefender 后，在 `config.yaml` 设置：
```yaml
anticheat:
  mode: "integrated"
paldefender:
  enabled: true
  address: "http://palworld:17993"
  token: "<在 PalDefender/RESTAPI/Tokens 签发的 Bearer Token>"
```

## 方式二：systemd 裸机

```bash
# 1. 在构建机（Ubuntu 22.04）上构建发布包
sudo apt install -y golang-go nodejs npm
bash scripts/build.sh          # 产出 dist/paladmin-<ver>-linux-amd64.tar.gz

# 2. 上传到目标服务器并解压
tar -xzf paladmin-*-linux-amd64.tar.gz
cd paladmin

# 3. 编辑配置
vim config.yaml               # 修改 web.password、rest.address、save.path、process.mode

# 4. 安装 systemd 服务（自动建用户、装 sav_cli Python 依赖）
sudo bash install.sh

sudo systemctl status paladmin
```

回档功能若要自动停/启游戏服：
- `process.mode: systemd`，`process.service: palworld`（参考 `deploy/palworld.service`）；
- 若不配置进程控制（`noop`），回档时需手动停服再操作。

## 关键配置项

| 配置 | 说明 |
|---|---|
| `web.password` | WebUI 密码（同时是 JWT 密钥），**必须修改** |
| `rest.address/password` | 游戏服官方 REST API（`:8212`）与 AdminPassword |
| `rcon.address/password` | RCON（`:25575`）与 AdminPassword |
| `save.path` | 游戏 `Saved` 目录（含 Level.sav） |
| `storage.path` | bbolt 数据库文件路径，默认 `./pst.db` |
| `process.mode` | `noop`/`systemd`/`docker`，控制回档时的停启服 |
| `anticheat.mode` | `external`（外置存档扫描）/ `integrated`（对接 PalDefender） |
| `paldefender.*` | PalDefender REST API 地址与 Token |

## 防火墙
```bash
sudo ufw allow 8080/tcp    # 或仅对内网/VPN 开放
# 游戏服 8211/UDP、RCON 25575、REST 8212 不应对公网开放
```

建议用 Nginx/Caddy 反代并启用 HTTPS。
