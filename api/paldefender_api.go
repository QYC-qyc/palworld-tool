package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

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

// createToken 一键生成 PalDefender REST API Token 文件（仅 Linux）。
func (p *palDefenderAPI) createToken(c *gin.Context) {
	if runtime.GOOS != "linux" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "仅支持 Linux"})
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	_ = c.ShouldBindJSON(&req)
	name := req.Name
	if name == "" {
		name = "PalAdmin"
	}

	st := p.detectAt("")
	if st.Win64Path == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "PalDefender 未安装（找不到游戏 Win64 目录）"})
		return
	}

	tokensDir := filepath.Join(st.Win64Path, "PalDefender", "RESTAPI", "Tokens")
	if err := os.MkdirAll(tokensDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	// 生成 48 字节随机 token
	buf := make([]byte, 48)
	if _, err := rand.Read(buf); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	token := base64.RawURLEncoding.EncodeToString(buf)

	// 文件名冲突时 name-2.json、name-3.json ...
	filename := name + ".json"
	for i := 2; ; i++ {
		if _, err := os.Stat(filepath.Join(tokensDir, filename)); os.IsNotExist(err) {
			break
		}
		filename = name + "-" + strconv.Itoa(i) + ".json"
	}

	payload := struct {
		Name        string   `json:"Name"`
		Token       string   `json:"Token"`
		Permissions []string `json:"Permissions"`
	}{
		Name:        name,
		Token:       token,
		Permissions: []string{"REST.*"},
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	if err := os.WriteFile(filepath.Join(tokensDir, filename), data, 0600); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	_ = audit.Add(db, "web", "paldefender_create_token", filename, "生成 REST API Token", "success")
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"token":      token,
		"token_file": filename,
		"tokens_dir": tokensDir,
	})
}

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
