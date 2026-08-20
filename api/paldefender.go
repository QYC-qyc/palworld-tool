package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	ghub "paladmin/internal/github"
)

const (
	palDefenderVersionAPI = "https://api.github.com/repos/Ultimeit/PalDefender/releases/latest"
	palDefenderDLL1       = "d3d9.dll"
	palDefenderDLL2       = "PalDefender.dll"
)

type palDefenderAPI struct{}

type pdStatus struct {
	Installed bool   `json:"installed"`
	Win64Path string `json:"win64_path"`
	D3d9Exists bool  `json:"d3d9_exists"`
	PdExists   bool  `json:"pd_exists"`
	GameDir   string `json:"game_dir"`
}

// status 检测 PalDefender 安装状态
func (p *palDefenderAPI) status(c *gin.Context) {
	st := p.detect()
	c.JSON(http.StatusOK, st)
}

// verify 验证指定游戏目录下的安装状态
func (p *palDefenderAPI) verify(c *gin.Context) {
	var req struct {
		GameDir string `json:"game_dir"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	st := p.detectAt(req.GameDir)
	c.JSON(http.StatusOK, st)
}

// pdInstallState PalDefender 安装任务状态
type pdInstallState struct {
	sync.Mutex
	running bool
	done    bool
	success bool
	errMsg  string
	percent int
	message string
	version string
}

var pdTask = &pdInstallState{}

// install 启动后台 PalDefender 下载安装
func (p *palDefenderAPI) install(c *gin.Context) {
	var req struct {
		GameDir string `json:"game_dir"`
	}
	_ = c.ShouldBindJSON(&req)

	pdTask.Lock()
	if pdTask.running {
		pdTask.Unlock()
		c.JSON(http.StatusOK, gin.H{"running": true})
		return
	}
	pdTask.running = true
	pdTask.done = false
	pdTask.success = false
	pdTask.errMsg = ""
	pdTask.percent = 0
	pdTask.message = "准备安装..."
	pdTask.version = ""
	pdTask.Unlock()

	go func() {
		defer func() {
			pdTask.Lock()
			pdTask.running = false
			pdTask.done = true
			pdTask.Unlock()
		}()

		setProgress := func(pct int, msg string) {
			pdTask.Lock()
			pdTask.percent = pct
			pdTask.message = msg
			pdTask.Unlock()
		}

		// 确定 Win64 目录
		gameDir := req.GameDir
		if gameDir == "" && gameAPI != nil {
			gameDir = gameAPI.mgr.ConfigValue().InstallDir
		}
		win64 := filepath.Join(gameDir, "Pal", "Binaries", "Win64")
		if err := os.MkdirAll(win64, 0755); err != nil {
			pdTask.Lock()
			pdTask.errMsg = "创建目录失败: " + err.Error()
			pdTask.Unlock()
			return
		}

		setProgress(10, "检查最新版本...")
		var rel struct {
			TagName string `json:"tag_name"`
			Assets  []struct {
				Name               string `json:"name"`
				BrowserDownloadURL string `json:"browser_download_url"`
			} `json:"assets"`
		}
		if err := ghub.FetchRelease(palDefenderVersionAPI, &rel); err != nil {
			pdTask.Lock()
			pdTask.errMsg = "检查版本失败: " + err.Error()
			pdTask.Unlock()
			return
		}

		dllNames := []string{palDefenderDLL1, palDefenderDLL2}
		for idx, name := range dllNames {
			setProgress(20+idx*35, fmt.Sprintf("下载 %s...", name))
			var dlURL string
			for _, a := range rel.Assets {
				if a.Name == name {
					dlURL = a.BrowserDownloadURL
					break
				}
			}
			if dlURL == "" {
				pdTask.Lock()
				pdTask.errMsg = "未找到 " + name
				pdTask.Unlock()
				return
			}
			dst := filepath.Join(win64, name)
			if err := ghub.DownloadToFile(dlURL, dst); err != nil {
				pdTask.Lock()
				pdTask.errMsg = "下载 " + name + " 失败: " + err.Error()
				pdTask.Unlock()
				return
			}
		}

		setProgress(95, "安装完成")
		pdTask.Lock()
		pdTask.success = true
		pdTask.percent = 100
		pdTask.message = fmt.Sprintf("PalDefender %s 安装成功", rel.TagName)
		pdTask.version = rel.TagName
		pdTask.Unlock()
	}()

	c.JSON(http.StatusOK, gin.H{"running": true, "message": "开始安装 PalDefender"})
}

// installStatus 返回 PalDefender 安装进度
func (p *palDefenderAPI) installStatus(c *gin.Context) {
	pdTask.Lock()
	defer pdTask.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"running": pdTask.running,
		"done":    pdTask.done,
		"success": pdTask.success,
		"error":   pdTask.errMsg,
		"percent": pdTask.percent,
		"message": pdTask.message,
		"version": pdTask.version,
	})
}

func (p *palDefenderAPI) detect() pdStatus {
	return p.detectAt("")
}

func (p *palDefenderAPI) detectAt(gameDir string) pdStatus {
	st := pdStatus{GameDir: gameDir}

	// 如果没指定游戏目录，从面板配置读取
	if gameDir == "" && gameAPI != nil {
		gameDir = gameAPI.mgr.ConfigValue().InstallDir
	}

	if gameDir != "" {
		win64 := filepath.Join(gameDir, "Pal", "Binaries", "Win64")
		if info, err := os.Stat(win64); err == nil && info.IsDir() {
			st.Win64Path = win64
			if _, err := os.Stat(filepath.Join(win64, palDefenderDLL1)); err == nil {
				st.D3d9Exists = true
			}
			if _, err := os.Stat(filepath.Join(win64, palDefenderDLL2)); err == nil {
				st.PdExists = true
			}
			st.Installed = st.D3d9Exists && st.PdExists
		}
	}

	return st
}

// uninstall 删除 PalDefender 的 DLL 文件
func (p *palDefenderAPI) uninstall(c *gin.Context) {
	var req struct {
		GameDir string `json:"game_dir"`
	}
	_ = c.ShouldBindJSON(&req)

	st := p.detectAt(req.GameDir)
	if st.Win64Path == "" {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "未找到安装目录，无需卸载"})
		return
	}

	removed := []string{}
	for _, name := range []string{palDefenderDLL1, palDefenderDLL2} {
		path := filepath.Join(st.Win64Path, name)
		if err := os.Remove(path); err == nil {
			removed = append(removed, name)
		}
	}
	// 删除 PalDefender 配置目录
	pdDir := filepath.Join(st.Win64Path, "PalDefender")
	_ = os.RemoveAll(pdDir)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("已卸载（删除 %s）", strings.Join(removed, ", ")),
	})
}
