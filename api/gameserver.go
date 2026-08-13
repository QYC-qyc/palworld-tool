package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"paladmin/internal/gamesrv"
	"paladmin/service"
)

type gameServerAPI struct {
	mgr *gamesrv.Manager
}

func newGameServerAPI() (*gameServerAPI, error) {
	return &gameServerAPI{mgr: gamesrv.NewManager()}, nil
}

// status 返回游戏服与 SteamCMD 状态
func (g *gameServerAPI) status(c *gin.Context) {
	st, err := g.mgr.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"available": true, "status": st})
}

// config 保存 SteamCMD 路径与安装目录等配置
func (g *gameServerAPI) saveConfig(c *gin.Context) {
	var cfg gamesrv.Config
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	g.mgr.SetConfig(cfg)
	// 持久化到数据库设置
	saveGameServerConfig(cfg)
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

// getConfig 读取已保存的配置
func (g *gameServerAPI) getConfig(c *gin.Context) {
	cfg := loadGameServerConfig()
	g.mgr.SetConfig(cfg)
	c.JSON(http.StatusOK, cfg)
}

// install 用 SteamCMD 安装/更新游戏服
func (g *gameServerAPI) install(c *gin.Context) {
	if err := g.mgr.Install(); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "已开始通过 SteamCMD 安装/更新，可在日志中查看进度",
	})
}

func (g *gameServerAPI) start(c *gin.Context) {
	if err := g.mgr.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "游戏服已启动"})
}

func (g *gameServerAPI) stop(c *gin.Context) {
	if err := g.mgr.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "游戏服已停止"})
}

func (g *gameServerAPI) restart(c *gin.Context) {
	if err := g.mgr.Restart(); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "游戏服已重启"})
}

func (g *gameServerAPI) restartImpl() error { return g.mgr.Restart() }

func (g *gameServerAPI) logs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"logs": g.mgr.Logs(200)})
}

// ---- 配置持久化（存数据库 settings）----

const (
	settingSteamCmd  = "gamesrv.steamcmd_path"
	settingInstall   = "gamesrv.install_dir"
	settingExtraArgs = "gamesrv.extra_args"
	settingGamePort  = "gamesrv.game_port"
)

func saveGameServerConfig(cfg gamesrv.Config) {
	if db == nil {
		return
	}
	_ = service.SetSettings(db, map[string]string{
		settingSteamCmd:  cfg.SteamCmdPath,
		settingInstall:   cfg.InstallDir,
		settingExtraArgs: cfg.ExtraArgs,
		settingGamePort:  cfg.GamePort,
	})
}

func loadGameServerConfig() gamesrv.Config {
	cfg := gamesrv.Config{
		SteamCmdPath: viper.GetString("gamesrv.steamcmd_path"),
		InstallDir:   viper.GetString("gamesrv.install_dir"),
		ExtraArgs:    viper.GetString("gamesrv.extra_args"),
		GamePort:     viper.GetString("gamesrv.game_port"),
	}
	if db == nil {
		return cfg
	}
	// 数据库中保存的值优先于环境变量
	if all, err := service.GetAllSettings(db); err == nil {
		if v, ok := all[settingSteamCmd]; ok && v != "" {
			cfg.SteamCmdPath = v
		}
		if v, ok := all[settingInstall]; ok && v != "" {
			cfg.InstallDir = v
		}
		if v, ok := all[settingExtraArgs]; ok {
			cfg.ExtraArgs = v
		}
		if v, ok := all[settingGamePort]; ok && v != "" {
			cfg.GamePort = v
		}
	}
	return cfg
}
