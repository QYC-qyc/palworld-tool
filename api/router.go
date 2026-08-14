package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"paladmin/internal/auth"
	"paladmin/service/audit"
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
		authGroup.POST("/player/:player_uid/kick", kickPlayer)
		authGroup.POST("/player/:player_uid/ban", banPlayer)
		authGroup.POST("/player/:player_uid/unban", unbanPlayer)
		authGroup.POST("/player/:player_uid/ipban", ipBanPlayer)
		authGroup.POST("/player/:player_uid/message", sendPlayerMessage)

		authGroup.PUT("/guild", putGuilds)

		authGroup.POST("/sync", syncData)

		authGroup.GET("/whitelist", listWhite)
		authGroup.POST("/whitelist", addWhite)
		authGroup.DELETE("/whitelist", removeWhite)
		authGroup.PUT("/whitelist", putWhite)

		authGroup.GET("/rcon", listRconCommand)
		authGroup.POST("/rcon", addRconCommand)
		authGroup.POST("/rcon/send", sendRconCommand)
		authGroup.PUT("/rcon/:uuid", putRconCommand)
		authGroup.DELETE("/rcon/:uuid", removeRconCommand)

		authGroup.GET("/backup", listBackups)
		authGroup.DELETE("/backup/:backup_id", deleteBackup)
		authGroup.POST("/backup/restore/:backup_id", restoreBackup)

		authGroup.GET("/banlist", listBans)
		authGroup.POST("/banip", banIP)
		authGroup.POST("/unbanip", unbanIP)

		// 游戏服管理
		if gameAPI != nil {
			authGroup.GET("/gameserver", gameAPI.status)
			authGroup.GET("/gameserver/config", gameAPI.getConfig)
			authGroup.PUT("/gameserver/config", gameAPI.saveConfig)
			authGroup.POST("/gameserver/verify", gameAPI.verify)
			authGroup.POST("/gameserver/install", gameAPI.install)
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
		authGroup.POST("/paldefender/install-wine", pdAPI.installWine)
		authGroup.GET("/paldefender/wine-status", pdAPI.wineInstallStatus)
		authGroup.POST("/paldefender/verify", pdAPI.verify)

		// 面板动态设置
		authGroup.GET("/settings", getSettings)
		authGroup.PUT("/settings", saveSettings)
		authGroup.POST("/settings/test-connection", testConnection)

		// 面板在线更新
		updaterAPI := &updaterAPI{}
		authGroup.GET("/updater/check", updaterAPI.check)
		authGroup.POST("/updater/do", updaterAPI.do)

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
