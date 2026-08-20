package api

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"paladmin/internal/palconfig"
	"paladmin/service"
)

// iniFallbackPath 当未配置安装目录时使用的兜底路径（Windows 原生）
const iniFallbackPath = `C:\PalServer\Pal\Saved\Config\WindowsServer\PalWorldSettings.ini`

// iniPath 解析 Windows 版游戏服的 PalWorldSettings.ini 完整路径。
// 面板直接管理 Windows 版服务端，配置固定在 WindowsServer 目录。
//  1. 环境变量 PALWORLD_INI_PATH 显式指定时优先；
//  2. 否则使用 <install_dir>/Pal/Saved/Config/WindowsServer/PalWorldSettings.ini。
func iniPath() string {
	if p := os.Getenv("PALWORLD_INI_PATH"); p != "" {
		return p
	}

	installDir := ""
	if gameAPI != nil {
		installDir = gameAPI.mgr.ConfigValue().InstallDir
	}
	if installDir == "" {
		installDir = os.Getenv("GAMESRV__INSTALL_DIR")
	}
	if installDir == "" {
		return iniFallbackPath
	}
	return filepath.Join(installDir, "Pal", "Saved", "Config", "WindowsServer", "PalWorldSettings.ini")
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

	// 网络相关项（端口、AdminPassword）同步到面板连接设置，避免两处配置不一致
	syncConnectionSettings(req.Settings)

	// 可选：重启游戏服使配置生效
	if req.Restart && gameAPI != nil {
		if err := gameAPI.restartImpl(); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success":   true,
				"message":   "配置已保存，但重启游戏服失败: " + err.Error(),
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

// syncConnectionSettings 将游戏配置中的网络项同步到面板连接设置，
// 避免「游戏配置」与「系统设置」两处的端口/密码不一致导致面板连不上游戏服。
// 面板本地启停游戏服时，REST 必然在本机：
//   - AdminPassword 同步到 rest.password
//   - RESTAPIPort 同步为 127.0.0.1 + 端口（地址为空、本机、或本机公网 IP 时）
func syncConnectionSettings(s map[string]string) {
	if db == nil {
		return
	}
	updates := map[string]string{}
	existing, _ := service.GetAllSettings(db)

	// 判断地址是否指向本机：空、localhost/127.0.0.1、容器名、或本机公网/内网 IP
	hostIsLocal := func(addr string) bool {
		if addr == "" {
			return true
		}
		low := strings.ToLower(addr)
		if strings.Contains(low, "127.0.0.1") || strings.Contains(low, "localhost") || strings.Contains(low, "palworld:") {
			return true
		}
		// 提取 host 部分
		host := addr
		if strings.HasPrefix(addr, "http://") {
			host = addr[7:]
		} else if strings.HasPrefix(addr, "https://") {
			host = addr[8:]
		}
		if idx := strings.IndexAny(host, ":/"); idx >= 0 {
			host = host[:idx]
		}
		return isLocalHost(host)
	}

	if adminPwd, ok := s["AdminPassword"]; ok && adminPwd != "" {
		updates[service.SettingRestPassword] = adminPwd
	}
	if port, ok := s["RESTAPIPort"]; ok && port != "" {
		if hostIsLocal(existing[service.SettingRestAddress]) {
			updates[service.SettingRestAddress] = "http://127.0.0.1:" + port
		}
	}

	if len(updates) > 0 {
		_ = service.SetSettings(db, updates)
	}
}

// isLocalHost 判断 host 是否为本机 IP（遍历网卡）
func isLocalHost(host string) bool {
	if host == "" {
		return false
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		if ipNet.IP.String() == host {
			return true
		}
	}
	return false
}
