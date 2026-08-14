package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var httpClient = &http.Client{Timeout: 5 * time.Minute}

const (
	palDefenderVersionAPI = "https://api.github.com/repos/Ultimeit/PalDefender/releases/latest"
	palDefenderDLL1      = "d3d9.dll"
	palDefenderDLL2      = "PalDefender.dll"
)

type palDefenderAPI struct{}

type pdStatus struct {
	Installed    bool   `json:"installed"`
	Win64Path   string `json:"win64_path"`
	WinePresent bool   `json:"wine_present"`
	WinePath    string `json:"wine_path"`
	WineVersion string `json:"wine_version"`
	D3d9Exists  bool   `json:"d3d9_exists"`
	PdExists    bool   `json:"pd_exists"`
	GameDir     string `json:"game_dir"`
}

// status 检测 PalDefender 安装状态和 Wine
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

// install 下载并安装 PalDefender DLL
func (p *palDefenderAPI) install(c *gin.Context) {
	var req struct {
		GameDir string `json:"game_dir"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if runtime.GOOS != "linux" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "PalDefender 自动安装仅支持 Linux"})
		return
	}

	st := p.detectAt(req.GameDir)
	if !st.WinePresent {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "未检测到 Wine，请先安装 Wine（apt install wine64）"})
		return
	}

	win64 := st.Win64Path
	if win64 == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "未找到 Pal/Binaries/Win64 目录，请确认游戏安装目录正确"})
		return
	}

	// 获取最新版本
	resp, err := httpClient.Get(palDefenderVersionAPI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "检查最新版本失败: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	var rel struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "解析版本信息失败: " + err.Error()})
		return
	}

	downloaded := 0
	for _, asset := range rel.Assets {
		if asset.Name != palDefenderDLL1 && asset.Name != palDefenderDLL2 {
			continue
		}
		dst := filepath.Join(win64, asset.Name)
		if err := downloadFile(asset.BrowserDownloadURL, dst); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: fmt.Sprintf("下载 %s 失败: %v", asset.Name, err),
			})
			return
		}
		downloaded++
	}

	if downloaded < 2 {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "未能下载全部 DLL 文件"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("PalDefender %s 已安装到 %s", rel.TagName, win64),
		"version": rel.TagName,
	})
}

func (p *palDefenderAPI) detect() pdStatus {
	return p.detectAt("")
}

func (p *palDefenderAPI) detectAt(gameDir string) pdStatus {
	st := pdStatus{GameDir: gameDir}

	// 检测 Wine
	if winePath, err := exec.LookPath("wine64"); err == nil {
		st.WinePresent = true
		st.WinePath = winePath
		if out, err := exec.Command("wine64", "--version").Output(); err == nil {
			st.WineVersion = strings.TrimSpace(string(out))
		}
	} else if winePath, err := exec.LookPath("wine"); err == nil {
		st.WinePresent = true
		st.WinePath = winePath
		if out, err := exec.Command("wine", "--version").Output(); err == nil {
			st.WineVersion = strings.TrimSpace(string(out))
		}
	}

	// 如果没指定游戏目录，从面板配置读取
	if gameDir == "" && gameAPI != nil {
		gameDir = gameAPI.mgr.ConfigValue().InstallDir
	}

	if gameDir != "" {
		win64 := filepath.Join(gameDir, "Pal", "Binaries", "Win64")
		if info, err := os.Stat(win64); err == nil && info.IsDir() {
			st.Win64Path = win64
			st.Installed = true
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

func downloadFile(url, dst string) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}
