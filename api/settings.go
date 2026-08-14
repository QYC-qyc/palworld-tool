package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"paladmin/internal/config"
	"paladmin/internal/tool"
	"paladmin/service"
	"paladmin/service/anticheat"
)

// testConnection 测试 REST / RCON 连通性
func testConnection(c *gin.Context) {
	var req struct {
		Type      string `json:"type"` // rest / rcon
		Address   string `json:"address"`
		Password  string `json:"password"`
		UseBase64 bool   `json:"use_base64"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	// 未输入密码时使用已保存的密码
	if req.Password == "" {
		switch req.Type {
		case "rest":
			req.Password = viper.GetString("rest.password")
		case "rcon":
			req.Password = viper.GetString("rcon.password")
		}
	}
	switch req.Type {
	case "rest":
		version, err := tool.TestRest(req.Address, req.Password)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "REST 连接成功", "version": version})
	case "rcon":
		err := tool.TestRcon(req.Address, req.Password, req.UseBase64)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "RCON 连接成功"})
	default:
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "type 必须是 rest 或 rcon"})
	}
}

// getSettings 返回全部动态配置（密码做脱敏：仅返回是否已设置）
func getSettings(c *gin.Context) {
	settings, err := service.GetAllSettings(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	// 敏感字段脱敏
	safe := map[string]interface{}{}
	for k, v := range settings {
		if isSecret(k) {
			if v != "" {
				safe[k] = "********"
				safe[k+"__set"] = "true"
			} else {
				safe[k] = ""
				safe[k+"__set"] = "false"
			}
		} else {
			safe[k] = v
		}
	}
	// 同时返回服务端口（静态，只读）
	safe["web.port"] = webPort()
	// 返回自动推导的存档目录（当 save.path 为空时，从游戏安装目录推导）
	if eff := tool.EffectiveSavePath(); eff != "" {
		safe["save.path_effective"] = filepath.Clean(eff)
	}
	c.JSON(http.StatusOK, safe)
}

// saveSettings 批量保存动态配置
func saveSettings(c *gin.Context) {
	// 用 interface{} 接收以兼容布尔、数字等非字符串值（前端开关等）
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	updates := map[string]string{}
	for k, v := range req {
		// 只允许已知键
		if !isEditableKey(k) {
			continue
		}
		str := toString(v)
		// 脱敏占位符不覆盖真实密码
		if isSecret(k) && str == "********" {
			continue
		}
		updates[k] = str
	}

	if err := service.SetSettings(db, updates); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	// 立即应用到 viper
	all, _ := service.GetAllSettings(db)
	config.ApplyToViper(all)

	_ = anticheat.AddAudit(db, "web", "settings_update", "", "更新面板动态配置", "success")
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

// toString 将任意 JSON 值转为字符串存储
func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		// 整数不显示小数点
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", t)
	}
}

func isSecret(k string) bool {
	return strings.Contains(k, "password") || strings.Contains(k, "token")
}

func isEditableKey(k string) bool {
	switch k {
	case service.SettingWebPassword, service.SettingRestAddress, service.SettingRestUsername,
		service.SettingRestPassword, service.SettingRconAddress, service.SettingRconPassword,
		service.SettingRconUseBase64, service.SettingSavePath, service.SettingProcessMode,
		service.SettingProcessService, service.SettingProcessContainer, service.SettingAnticheatMode,
		service.SettingAnticheatEnabled, service.SettingPaldefenderAddr, service.SettingPaldefenderToken:
		return true
	}
	return false
}
