package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"paladmin/internal/palconfig"
)

// 宿主机游戏配置文件路径（游戏服容器把数据写到 ./game）
func iniPath() string {
	// 默认位置，可通过环境变量覆盖
	if p := os.Getenv("PALWORLD_INI_PATH"); p != "" {
		return p
	}
	return "/www/palworld-tool/game/Pal/Saved/Config/LinuxServer/PalWorldSettings.ini"
}

type gameSettingsAPI struct{}

// schema 返回配置项定义
func (g *gameSettingsAPI) schema(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"fields":  palconfig.Schema(),
		"iniPath": iniPath(),
	})
}

// get 读取当前配置
func (g *gameSettingsAPI) get(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = iniPath()
	}
	settings := map[string]string{}
	if b, err := os.ReadFile(path); err == nil {
		settings = palconfig.Parse(string(b))
	} else if !os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "读取配置失败: " + err.Error()})
		return
	}
	// 用默认值补全未设置项
	for _, f := range palconfig.Schema() {
		if _, ok := settings[f.Key]; !ok {
			settings[f.Key] = f.Default
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"settings": settings,
		"path":     path,
		"exists":   fileExists(path),
	})
}

// save 保存配置并提示需重启游戏服
func (g *gameSettingsAPI) save(c *gin.Context) {
	var req struct {
		Path     string            `json:"path"`
		Settings map[string]string `json:"settings"`
		Restart  bool              `json:"restart"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	if req.Path == "" {
		req.Path = iniPath()
	}

	// 校验
	for _, f := range palconfig.Schema() {
		if v, ok := req.Settings[f.Key]; ok && v != "" {
			if err := palconfig.ValidateValue(f, v); err != nil {
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
				return
			}
		}
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(req.Path), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "创建目录失败: " + err.Error()})
		return
	}

	if err := palconfig.SaveFile(req.Path, req.Settings); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "写入失败: " + err.Error()})
		return
	}

	// 可选：重启游戏服使配置生效
	if req.Restart && gameAPI != nil {
		if err := gameAPI.restartImpl(); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success":  true,
				"message":  "配置已保存，但重启游戏服失败: " + err.Error(),
				"restarted": false,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "配置已保存，游戏服重启中", "restarted": true})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置已保存，重启游戏服后生效",
	})
}

func (g *gameSettingsAPI) raw(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = iniPath()
	}
	b, err := os.ReadFile(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	// 脱敏密码
	content := string(b)
	content = maskPasswords(content)
	c.JSON(http.StatusOK, gin.H{"content": content, "path": path})
}

// maskPasswords 把输出中的密码值替换为 ***
func maskPasswords(s string) string {
	for _, key := range []string{"AdminPassword", "ServerPassword", "RESTAPIPassword"} {
		idx := strings.Index(s, key+"=")
		if idx < 0 {
			continue
		}
		s = s[:idx] + key + `="***"` + s[idx+len(key)+1:]
	}
	return s
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
