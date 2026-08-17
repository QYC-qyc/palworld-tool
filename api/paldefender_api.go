package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"paladmin/internal/paldefender"
	"paladmin/service/audit"
)

// pdLoad 加载 PalDefender 客户端，未配置时直接写入 400 错误并返回 nil。
func pdLoad(c *gin.Context) *paldefender.Client {
	client, err := paldefender.Load(db)
	if err != nil {
		if errors.Is(err, paldefender.ErrNotConfigured) {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "PalDefender API 未配置，请在设置中启用并填写 Token"})
			return nil
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return nil
	}
	return client
}

// pdFail 把 PalDefender 客户端错误归一为 502 响应。
func pdFail(c *gin.Context, err error) {
	c.JSON(http.StatusBadGateway, ErrorResponse{Error: err.Error()})
}

// pdRaw 透传 PalDefender 返回的原始 JSON。
func pdRaw(c *gin.Context, raw []byte) {
	c.Data(http.StatusOK, "application/json", raw)
}

// --- 连通性 ---

func (p *palDefenderAPI) apiVersion(c *gin.Context) {
	client := pdLoad(c)
	if client == nil {
		return
	}
	raw, err := client.Version()
	if err != nil {
		pdFail(c, err)
		return
	}
	pdRaw(c, raw)
}

// --- 玩家 ---

func (p *palDefenderAPI) apiListPlayers(c *gin.Context) {
	client := pdLoad(c)
	if client == nil {
		return
	}
	raw, err := client.ListPlayers()
	if err != nil {
		pdFail(c, err)
		return
	}
	pdRaw(c, raw)
}

func (p *palDefenderAPI) apiGetPlayer(c *gin.Context) {
	client := pdLoad(c)
	if client == nil {
		return
	}
	raw, err := client.GetPlayer(c.Param("id"))
	if err != nil {
		pdFail(c, err)
		return
	}
	pdRaw(c, raw)
}

func (p *palDefenderAPI) apiKick(c *gin.Context) {
	client := pdLoad(c)
	if client == nil {
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	id := c.Param("id")
	raw, err := client.Kick(id, req.Reason)
	if err != nil {
		pdFail(c, err)
		return
	}
	_ = audit.Add(db, "web", "paldefender_kick", id, req.Reason, "success")
	pdRaw(c, raw)
}

func (p *palDefenderAPI) apiBan(c *gin.Context) {
	client := pdLoad(c)
	if client == nil {
		return
	}
	var req struct {
		Reason string `json:"reason"`
		IP     bool   `json:"ip"`
	}
	_ = c.ShouldBindJSON(&req)
	id := c.Param("id")
	raw, err := client.Ban(id, req.Reason, req.IP)
	if err != nil {
		pdFail(c, err)
		return
	}
	_ = audit.Add(db, "web", "paldefender_ban", id, req.Reason, "success")
	pdRaw(c, raw)
}

func (p *palDefenderAPI) apiUnban(c *gin.Context) {
	client := pdLoad(c)
	if client == nil {
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	userID := c.Param("user_id")
	raw, err := client.Unban(userID, req.Reason)
	if err != nil {
		pdFail(c, err)
		return
	}
	_ = audit.Add(db, "web", "paldefender_unban", userID, req.Reason, "success")
	pdRaw(c, raw)
}

func (p *palDefenderAPI) apiBanIP(c *gin.Context) {
	client := pdLoad(c)
	if client == nil {
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	ip := c.Param("ip")
	raw, err := client.BanIP(ip, req.Reason)
	if err != nil {
		pdFail(c, err)
		return
	}
	_ = audit.Add(db, "web", "paldefender_banip", ip, req.Reason, "success")
	pdRaw(c, raw)
}

func (p *palDefenderAPI) apiUnbanIP(c *gin.Context) {
	client := pdLoad(c)
	if client == nil {
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	ip := c.Param("ip")
	raw, err := client.UnbanIP(ip, req.Reason)
	if err != nil {
		pdFail(c, err)
		return
	}
	_ = audit.Add(db, "web", "paldefender_unbanip", ip, req.Reason, "success")
	pdRaw(c, raw)
}

// --- 封禁列表 ---

func (p *palDefenderAPI) apiBanlist(c *gin.Context) {
	client := pdLoad(c)
	if client == nil {
		return
	}
	raw, err := client.Banlist(c.Request.URL.Query())
	if err != nil {
		pdFail(c, err)
		return
	}
	pdRaw(c, raw)
}

// --- 消息 ---

func (p *palDefenderAPI) apiBroadcast(c *gin.Context) {
	client := pdLoad(c)
	if client == nil {
		return
	}
	var req struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Message == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "message 不能为空"})
		return
	}
	raw, err := client.Broadcast(req.Message)
	if err != nil {
		pdFail(c, err)
		return
	}
	_ = audit.Add(db, "web", "paldefender_broadcast", "", req.Message, "success")
	pdRaw(c, raw)
}

func (p *palDefenderAPI) apiAlert(c *gin.Context) {
	client := pdLoad(c)
	if client == nil {
		return
	}
	var req struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Message == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "message 不能为空"})
		return
	}
	raw, err := client.Alert(req.Message)
	if err != nil {
		pdFail(c, err)
		return
	}
	_ = audit.Add(db, "web", "paldefender_alert", "", req.Message, "success")
	pdRaw(c, raw)
}

func (p *palDefenderAPI) apiMessage(c *gin.Context) {
	client := pdLoad(c)
	if client == nil {
		return
	}
	var req struct {
		UserID   string `json:"userId"`
		Message  string `json:"message"`
		SendType string `json:"sendType"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.UserID == "" || req.Message == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "userId 和 message 不能为空"})
		return
	}
	raw, err := client.SendPlayerMessage(req.UserID, req.Message, req.SendType)
	if err != nil {
		pdFail(c, err)
		return
	}
	_ = audit.Add(db, "web", "paldefender_message", req.UserID, req.Message, "success")
	pdRaw(c, raw)
}

// --- 公会/据点 ---

func (p *palDefenderAPI) apiListGuilds(c *gin.Context) {
	client := pdLoad(c)
	if client == nil {
		return
	}
	raw, err := client.ListGuilds()
	if err != nil {
		pdFail(c, err)
		return
	}
	pdRaw(c, raw)
}

func (p *palDefenderAPI) apiGetGuild(c *gin.Context) {
	client := pdLoad(c)
	if client == nil {
		return
	}
	raw, err := client.GetGuild(c.Param("id"))
	if err != nil {
		pdFail(c, err)
		return
	}
	pdRaw(c, raw)
}

func (p *palDefenderAPI) apiDeleteBase(c *gin.Context) {
	client := pdLoad(c)
	if client == nil {
		return
	}
	id := c.Param("id")
	raw, err := client.DeleteBase(id)
	if err != nil {
		pdFail(c, err)
		return
	}
	_ = audit.Add(db, "web", "paldefender_deletebase", id, "删除据点", "success")
	pdRaw(c, raw)
}

// --- 工具 ---

func (p *palDefenderAPI) apiReloadConfig(c *gin.Context) {
	client := pdLoad(c)
	if client == nil {
		return
	}
	raw, err := client.ReloadConfig()
	if err != nil {
		pdFail(c, err)
		return
	}
	_ = audit.Add(db, "web", "paldefender_reload", "", "热重载配置", "success")
	pdRaw(c, raw)
}
