package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"palworld-panel/internal/auth"
	"palworld-panel/service/audit"
)

type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// RegisterRouter 注册全部路由
func RegisterRouter(r *gin.Engine) {
	r.Use(gin.Recovery())

	// 公开接口：初始化向导与登录
	r.GET("/api/setup/status", setupStatus)
	r.POST("/api/setup", setup)
	r.POST("/api/login", loginHandler)

	apiGroup := r.Group("/api")

	anon := apiGroup.Group("")
	{
		anon.GET("/server", getServer)
		anon.GET("/server/metrics", getServerMetrics)
		anon.GET("/player", listPlayers)
		anon.GET("/player/:player_uid", getPlayer)
		anon.GET("/online_player", listOnlinePlayers)
		anon.GET("/guild", listGuilds)
		anon.GET("/guild/:admin_player_uid", getGuild)
	}

	authGroup := apiGroup.Group("")
	authGroup.Use(auth.JWTAuthMiddleware())
	{
		authGroup.POST("/server/broadcast", publishBroadcast)
		authGroup.POST("/server/shutdown", shutdownServer)

		authGroup.PUT("/player", putPlayers)
		authGroup.POST("/player/:player_uid/message", sendPlayerMessage)

		authGroup.PUT("/guild", putGuilds)

		authGroup.POST("/sync", syncData)

		authGroup.GET("/whitelist", listWhite)
		authGroup.POST("/whitelist", addWhite)
		authGroup.DELETE("/whitelist", removeWhite)
		authGroup.PUT("/whitelist", putWhite)

		authGroup.GET("/backup", listBackups)
		authGroup.DELETE("/backup/:backup_id", deleteBackup)
		authGroup.POST("/backup/restore/:backup_id", restoreBackup)

		// 游戏服管理
		if gameAPI != nil {
			authGroup.GET("/gameserver", gameAPI.status)
			authGroup.GET("/gameserver/config", gameAPI.getConfig)
			authGroup.PUT("/gameserver/config", gameAPI.saveConfig)
			authGroup.POST("/gameserver/verify", gameAPI.verify)
			authGroup.POST("/gameserver/install", gameAPI.install)
			authGroup.POST("/gameserver/install-steamcmd", gameAPI.installSteamCMD)
			authGroup.POST("/gameserver/start", gameAPI.start)
			authGroup.POST("/gameserver/stop", gameAPI.stop)
			authGroup.POST("/gameserver/restart", gameAPI.restart)
			authGroup.GET("/gameserver/logs", gameAPI.logs)
		}

		// 游戏服 .ini 配置
		authGroup.GET("/gamesettings/schema", gameSettings.schema)
		authGroup.GET("/gamesettings", gameSettings.get)
		authGroup.PUT("/gamesettings", gameSettings.save)
		authGroup.GET("/gamesettings/raw", gameSettings.raw)

		// PalDefender 集成管理
		pdAPI := &palDefenderAPI{}
		authGroup.GET("/paldefender/status", pdAPI.status)
		authGroup.POST("/paldefender/install", pdAPI.install)
		authGroup.GET("/paldefender/install-status", pdAPI.installStatus)
		authGroup.POST("/paldefender/uninstall", pdAPI.uninstall)
		authGroup.POST("/paldefender/verify", pdAPI.verify)
		authGroup.POST("/paldefender/create-token", pdAPI.createToken)

		// PalDefender REST API 代理（前缀 /paldefender/api）
		pdAPIGroup := authGroup.Group("/paldefender/api")
		{
			pdAPIGroup.GET("/version", pdAPI.apiVersion)

			pdAPIGroup.GET("/players", pdAPI.apiListPlayers)
			pdAPIGroup.GET("/players/:id", pdAPI.apiGetPlayer)
			pdAPIGroup.POST("/players/:id/kick", pdAPI.apiKick)
			pdAPIGroup.POST("/players/:id/ban", pdAPI.apiBan)
			pdAPIGroup.POST("/unban/:user_id", pdAPI.apiUnban)
			pdAPIGroup.POST("/banip/:ip", pdAPI.apiBanIP)
			pdAPIGroup.POST("/unbanip/:ip", pdAPI.apiUnbanIP)
			pdAPIGroup.GET("/banlist", pdAPI.apiBanlist)

			pdAPIGroup.POST("/broadcast", pdAPI.apiBroadcast)
			pdAPIGroup.POST("/alert", pdAPI.apiAlert)
			pdAPIGroup.POST("/message", pdAPI.apiMessage)

			pdAPIGroup.GET("/guilds", pdAPI.apiListGuilds)
			pdAPIGroup.GET("/guilds/:id", pdAPI.apiGetGuild)
			pdAPIGroup.POST("/deletebase/:id", pdAPI.apiDeleteBase)

			pdAPIGroup.POST("/reload-config", pdAPI.apiReloadConfig)
		}

		// 面板动态设置
		authGroup.GET("/settings", getSettings)
		authGroup.PUT("/settings", saveSettings)
		authGroup.POST("/settings/test-connection", testConnection)

		// 面板在线更新
		updaterAPI := &updaterAPI{}
		authGroup.GET("/updater/check", updaterAPI.check)
		authGroup.POST("/updater/do", updaterAPI.do)
		authGroup.GET("/updater/status", updaterAPI.status)

		// 容器内一键自更新（compose pull + up）
		authGroup.GET("/self-update/status", selfUpdateStatus)
		authGroup.POST("/self-update/do", selfUpdateDo)

		// 审计日志
		authGroup.GET("/audit", func(c *gin.Context) {
			limit := 100
			records, err := audit.List(db, limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
				return
			}
			c.JSON(http.StatusOK, records)
		})
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
