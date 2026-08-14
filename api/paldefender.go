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
	"sync"
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

	// 自动确定或创建 Win64 目录
	win64 := st.Win64Path
	if win64 == "" {
		gameDir := req.GameDir
		if gameDir == "" && gameAPI != nil {
			gameDir = gameAPI.mgr.ConfigValue().InstallDir
		}
		if gameDir == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "未配置游戏安装目录"})
			return
		}
		win64 = filepath.Join(gameDir, "Pal", "Binaries", "Win64")
		if err := os.MkdirAll(win64, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "创建 Win64 目录失败: " + err.Error()})
			return
		}
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

	// 下载 DLL，通过国内镜像加速
	dlMirrors := []string{
		"https://ghfast.top/",
		"https://gh-proxy.com/",
		"https://ghproxy.net/",
		"",
	}
	downloaded := 0
	for _, asset := range rel.Assets {
		if asset.Name != palDefenderDLL1 && asset.Name != palDefenderDLL2 {
			continue
		}
		dst := filepath.Join(win64, asset.Name)
		var lastErr error
		for _, prefix := range dlMirrors {
			url := prefix + asset.BrowserDownloadURL
			if err := downloadFile(url, dst); err == nil {
				lastErr = nil
				break
			} else {
				lastErr = err
			}
		}
		if lastErr != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: fmt.Sprintf("下载 %s 失败: %v", asset.Name, lastErr),
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

// wineInstallState Wine 安装任务状态
type wineInstallState struct {
	running   bool
	done      bool
	success   bool
	log       strings.Builder
	errMsg    string
	mu        sync.Mutex
}

var wineTask = &wineInstallState{}

// installWine 后台执行 Wine 安装，立即返回
func (p *palDefenderAPI) installWine(c *gin.Context) {
	if runtime.GOOS != "linux" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "仅支持 Linux"})
		return
	}

	wineTask.mu.Lock()
	if wineTask.running {
		wineTask.mu.Unlock()
		c.JSON(http.StatusOK, gin.H{"running": true, "message": "正在安装中..."})
		return
	}
	wineTask.running = true
	wineTask.done = false
	wineTask.success = false
	wineTask.errMsg = ""
	wineTask.log.Reset()
	wineTask.mu.Unlock()

	go func() {
		defer func() {
			wineTask.mu.Lock()
			wineTask.running = false
			wineTask.done = true
			wineTask.mu.Unlock()
		}()

		commands := [][]string{
			{"dpkg", "--add-architecture", "i386"},
			{"apt-get", "update", "-y"},
			{"apt-get", "install", "-y", "wine64"},
		}

		for _, args := range commands {
			wineTask.mu.Lock()
			wineTask.log.WriteString("$ " + strings.Join(args, " ") + "\n")
			wineTask.mu.Unlock()

			cmd := exec.Command(args[0], args[1:]...)
			setSysProcAttr(cmd)
			stdout, _ := cmd.StdoutPipe()
			cmd.Stderr = cmd.Stdout
			if err := cmd.Start(); err != nil {
				wineTask.mu.Lock()
				wineTask.errMsg = "启动失败: " + err.Error()
				wineTask.mu.Unlock()
				return
			}
			buf := make([]byte, 4096)
			for {
				n, err := stdout.Read(buf)
				if n > 0 {
					wineTask.mu.Lock()
					wineTask.log.Write(buf[:n])
					wineTask.mu.Unlock()
				}
				if err != nil {
					break
				}
			}
			if err := cmd.Wait(); err != nil {
				wineTask.mu.Lock()
				wineTask.errMsg = fmt.Sprintf("执行 %s 失败: %v", args[0], err)
				wineTask.mu.Unlock()
				return
			}
		}

		if out, err := exec.Command("wine64", "--version").Output(); err == nil {
			wineTask.mu.Lock()
			wineTask.success = true
			wineTask.log.WriteString("\n安装完成: " + strings.TrimSpace(string(out)) + "\n")
			wineTask.mu.Unlock()
		} else {
			wineTask.mu.Lock()
			wineTask.success = true
			wineTask.log.WriteString("\nWine 安装完成，请刷新状态\n")
			wineTask.mu.Unlock()
		}
	}()

	c.JSON(http.StatusOK, gin.H{"running": true, "message": "开始安装 Wine..."})
}

// wineInstallStatus 返回 Wine 安装进度
func (p *palDefenderAPI) wineInstallStatus(c *gin.Context) {
	wineTask.mu.Lock()
	defer wineTask.mu.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"running": wineTask.running,
		"done":    wineTask.done,
		"success": wineTask.success,
		"error":   wineTask.errMsg,
		"log":     wineTask.log.String(),
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
