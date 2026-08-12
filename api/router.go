package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"paladmin/internal/auth"
)

type SuccessResponse struct {
	Success bool `json:"success"`
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

		// 面板动态设置
		authGroup.GET("/settings", getSettings)
		authGroup.PUT("/settings", saveSettings)

		// 反作弊
		authGroup.GET("/anticheat/alert", listAlerts)
		authGroup.GET("/anticheat/alert/:id", getAlert)
		authGroup.POST("/anticheat/alert/:id/action", alertAction)
		authGroup.GET("/anticheat/rule", listRules)
		authGroup.PUT("/anticheat/rule/:id", updateRule)
		authGroup.POST("/anticheat/scan", runScan)
		authGroup.GET("/anticheat/stats", acStats)
		authGroup.GET("/anticheat/audit", listAudit)
		authGroup.POST("/anticheat/reload", reloadAC)
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
