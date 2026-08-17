package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"paladmin/internal/database"
	"paladmin/internal/detect"
	"paladmin/internal/paldefender"
	"paladmin/internal/tool"
	"paladmin/service"
	"paladmin/service/audit"
)

func listPlayers(c *gin.Context) {
	players, err := service.ListPlayers(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, players)
}

func getPlayer(c *gin.Context) {
	uid := c.Param("player_uid")
	p, err := service.GetPlayer(db, uid)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "玩家不存在"})
		return
	}
	c.JSON(http.StatusOK, p)
}

func listOnlinePlayers(c *gin.Context) {
	online, err := tool.ShowPlayers()
	if err != nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	c.JSON(http.StatusOK, online)
}

// putPlayers sav_cli 回调用
func putPlayers(c *gin.Context) {
	var players []database.Player
	if err := c.ShouldBindJSON(&players); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	if err := service.PutPlayers(db, players); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	// 触发轻量存档反作弊检测
	go func() {
		cfg := detect.DefaultConfig()
		detect.RunSaveCheck(db, cfg, players)
	}()
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

func kickPlayer(c *gin.Context) {
	uid := c.Param("player_uid")
	p, err := service.GetPlayer(db, uid)
	if err != nil || p.SteamId == "" {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "未找到该玩家或缺少 SteamID"})
		return
	}
	if err := tool.KickPlayer(p.SteamId); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	_ = audit.Add(db, "web", "kick", p.SteamId, "手动踢出", "success")
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

func banPlayer(c *gin.Context) {
	uid := c.Param("player_uid")
	p, err := service.GetPlayer(db, uid)
	if err != nil || p.SteamId == "" {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "未找到该玩家或缺少 SteamID"})
		return
	}
	if err := tool.BanPlayer(p.SteamId); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	_ = service.AddBan(db, database.BanRecord{
		Type: database.BanUser, Identifier: "steam_" + p.SteamId,
		Reason: "管理员手动封禁", Issuer: "web",
	})
	_ = audit.Add(db, "web", "ban", p.SteamId, "手动封禁", "success")
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

func unbanPlayer(c *gin.Context) {
	uid := c.Param("player_uid")
	p, err := service.GetPlayer(db, uid)
	if err != nil || p.SteamId == "" {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "未找到该玩家或缺少 SteamID"})
		return
	}
	if err := tool.UnBanPlayer(p.SteamId); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	_ = service.RemoveBan(db, database.BanUser, "steam_"+p.SteamId)
	_ = audit.Add(db, "web", "unban", p.SteamId, "手动解封", "success")
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

func ipBanPlayer(c *gin.Context) {
	uid := c.Param("player_uid")
	p, err := service.GetPlayer(db, uid)
	if err != nil || p.Ip == "" {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "未找到该玩家或缺少 IP"})
		return
	}
	_ = service.AddBan(db, database.BanRecord{
		Type: database.BanIP, Identifier: p.Ip,
		Reason: "管理员 IP 封禁", Issuer: "web",
	})
	_ = tool.BanPlayer(p.SteamId)
	_ = audit.Add(db, "web", "ipban", p.Ip, "手动IP封禁", "success")
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

type messageRequest struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func sendPlayerMessage(c *gin.Context) {
	uid := c.Param("player_uid")
	var req messageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	p, err := service.GetPlayer(db, uid)
	if err != nil || p.SteamId == "" {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "未找到该玩家"})
		return
	}
	// 私聊通过 PalDefender REST API 实现
	client, err := paldefender.Load(db)
	if err != nil {
		if errors.Is(err, paldefender.ErrNotConfigured) {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "私聊需要启用并配置 PalDefender"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	userID := "steam_" + p.SteamId
	if _, err := client.SendPlayerMessage(userID, req.Message, "PlayerChat"); err != nil {
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: err.Error()})
		return
	}
	_ = audit.Add(db, "web", "message", userID, req.Message, "success")
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}
