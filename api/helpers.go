package api

import (
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// inContainer 检测面板是否运行在容器内（存在 /.dockerenv 或显式 PALADIN_CONTAINER=1）。
// 容器内禁用面板自更新等只对宿主机二进制部署有意义的功能。
func inContainer() bool {
	if os.Getenv("PALADIN_CONTAINER") == "1" {
		return true
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return false
}

func webPassword() string {
	return viper.GetString("web.password")
}

func webPort() int {
	return viper.GetInt("web.port")
}

func queryInt(c *gin.Context, key string, def int) int {
	s := c.Query(key)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}
