package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

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
