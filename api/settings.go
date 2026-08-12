package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"paladmin/internal/config"
	"paladmin/service"
	"paladmin/service/anticheat"
)

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
	c.JSON(http.StatusOK, safe)
}

// saveSettings 批量保存动态配置
func saveSettings(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	updates := map[string]string{}
	for k, v := range req {
		// 脱敏占位符不覆盖真实密码
		if isSecret(k) && v == "********" {
			continue
		}
		// 只允许已知键
		if !isEditableKey(k) {
			continue
		}
		updates[k] = v
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

func isSecret(k string) bool {
	return strings.Contains(k, "password") || strings.Contains(k, "token")
}

func isEditableKey(k string) bool {
	switch k {
	case service.SettingWebPassword, service.SettingRestAddress, service.SettingRestUsername,
		service.SettingRestPassword, service.SettingRconAddress, service.SettingRconPassword,
		service.SettingRconUseBase64, service.SettingSavePath, service.SettingProcessMode,
		service.SettingProcessService, service.SettingProcessContainer, service.SettingAnticheatMode,
		service.SettingAnticheatEnabled, service.SettingPaldefenderAddr, service.SettingPaldefenderToken,
		service.SettingNotifyWebhook, service.SettingNotifyType:
		return true
	}
	return false
}
