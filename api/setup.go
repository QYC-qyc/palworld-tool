package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"paladmin/internal/config"
	"paladmin/service"
)

// setupStatus 返回系统是否已初始化（无需认证）
func setupStatus(c *gin.Context) {
	pwd := service.GetSetting(db, service.SettingWebPassword)
	c.JSON(http.StatusOK, gin.H{
		"initialized": pwd != "",
	})
}

type setupRequest struct {
	WebPassword    string `json:"web_password"`
	RestAddress    string `json:"rest_address"`
	RestPassword   string `json:"rest_password"`
	RconAddress    string `json:"rcon_address"`
	RconPassword   string `json:"rcon_password"`
	ServerName     string `json:"server_name"`
}

// setup 首次初始化：设置管理员密码和游戏连接信息（无需登录，但仅在未初始化时允许）
func setup(c *gin.Context) {
	// 已初始化则拒绝
	if service.GetSetting(db, service.SettingWebPassword) != "" {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "系统已初始化"})
		return
	}

	var req setupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	if req.WebPassword == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "管理员密码不能为空"})
		return
	}

	updates := map[string]string{
		service.SettingWebPassword: req.WebPassword,
	}
	if req.RestAddress != "" {
		updates[service.SettingRestAddress] = req.RestAddress
	}
	if req.RestPassword != "" {
		updates[service.SettingRestPassword] = req.RestPassword
	}
	if req.RconAddress != "" {
		updates[service.SettingRconAddress] = req.RconAddress
	}
	if req.RconPassword != "" {
		updates[service.SettingRconPassword] = req.RconPassword
	}

	if err := service.SetSettings(db, updates); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	// 立即生效
	all, _ := service.GetAllSettings(db)
	config.ApplyToViper(all)

	// 初始化后直接签发登录令牌
	token, err := loginToken(req.WebPassword)
	if err != nil {
		c.JSON(http.StatusOK, SuccessResponse{Success: true})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "token": token})
}
