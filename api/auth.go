package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"paladmin/internal/auth"
	"paladmin/service"
)

type loginRequest struct {
	Password string `json:"password"`
}

func loginHandler(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Password = c.PostForm("password")
	}
	// 密码以初始化向导设置的为准（存数据库）。未初始化则拒绝登录。
	expected := service.GetSetting(db, service.SettingWebPassword)
	if expected == "" {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "系统未初始化"})
		return
	}
	if req.Password != expected {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "密码错误"})
		return
	}
	token, err := loginToken(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// loginToken 用当前密码签发 JWT（校验由调用方完成）
func loginToken(password string) (string, error) {
	return auth.GenerateTokenForUser("admin")
}
