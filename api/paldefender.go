package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"paladmin/internal/tool"
	"paladmin/service/anticheat"
)

// pdStatus 检查 PalDefender 连接状态
func pdStatus(c *gin.Context) {
	pd := tool.NewPalDefender()
	if !pd.Available() {
		c.JSON(http.StatusOK, gin.H{"available": false, "message": "未配置 PalDefender（paldefender.address/token）"})
		return
	}
	version, err := pd.Version()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"available": true, "connected": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"available": true, "connected": true, "version": version})
}

// pdBanlist 读取 PalDefender 封禁列表
func pdBanlist(c *gin.Context) {
	pd := tool.NewPalDefender()
	if !pd.Available() {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "PalDefender 未配置"})
		return
	}
	list, err := pd.Banlist(c.Request.URL.RawQuery)
	if err != nil {
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// pdPlayers 读取 PalDefender 玩家列表
func pdPlayers(c *gin.Context) {
	pd := tool.NewPalDefender()
	if !pd.Available() {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "PalDefender 未配置"})
		return
	}
	players, err := pd.Players()
	if err != nil {
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, players)
}

// pdReload 让 PalDefender 重载配置
func pdReload(c *gin.Context) {
	pd := tool.NewPalDefender()
	if err := pd.ReloadConfig(); err != nil {
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: err.Error()})
		return
	}
	_ = anticheat.AddAudit(db, "web", "paldefender_reload", "", "重载 PalDefender 配置", "success")
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}
