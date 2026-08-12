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
	// 优先用数据库中动态设置的密码
	expected := service.GetSetting(db, service.SettingWebPassword)
	if expected == "" {
		expected = webPassword()
	}
	if expected != "" && req.Password != expected {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "密码错误"})
		return
	}
	token, err := auth.GenerateTokenForUser("admin")
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}
