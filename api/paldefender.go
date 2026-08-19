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

// pdInstallState PalDefender 安装任务状态
type pdInstallState struct {
	sync.Mutex
	running  bool
	done     bool
	success  bool
	errMsg   string
	percent  int
	message  string
	version  string
}

var pdTask = &pdInstallState{}

// install 启动后台 PalDefender 下载安装
func (p *palDefenderAPI) install(c *gin.Context) {
	var req struct {
		GameDir string `json:"game_dir"`
	}
	_ = c.ShouldBindJSON(&req)

	if runtime.GOOS != "linux" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "仅支持 Linux"})
		return
	}

	st := p.detectAt(req.GameDir)
	if !st.WinePresent {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "未检测到 Wine，请先安装"})
		return
	}

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
		win64 := st.Win64Path
		if win64 == "" {
			gameDir := req.GameDir
			if gameDir == "" && gameAPI != nil {
				gameDir = gameAPI.mgr.ConfigValue().InstallDir
			}
			win64 = filepath.Join(gameDir, "PalServer-Win", "Pal", "Binaries", "Win64")
			if err := os.MkdirAll(win64, 0755); err != nil {
				pdTask.Lock()
				pdTask.errMsg = "创建目录失败: " + err.Error()
				pdTask.Unlock()
				return
			}
		}

		setProgress(10, "检查最新版本...")
		resp, err := httpClient.Get(palDefenderVersionAPI)
		if err != nil {
			pdTask.Lock()
			pdTask.errMsg = "检查版本失败: " + err.Error()
			pdTask.Unlock()
			return
		}
		var rel struct {
			TagName string `json:"tag_name"`
			Assets  []struct {
				Name               string `json:"name"`
				BrowserDownloadURL string `json:"browser_download_url"`
			} `json:"assets"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
			resp.Body.Close()
			pdTask.Lock()
			pdTask.errMsg = "解析版本失败: " + err.Error()
			pdTask.Unlock()
			return
		}
		resp.Body.Close()

		dlMirrors := []string{
			"https://ghfast.top/",
			"https://gh-proxy.com/",
			"https://ghproxy.net/",
			"",
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
			var ok bool
			for _, prefix := range dlMirrors {
				if err := downloadFile(prefix+dlURL, dst); err == nil {
					ok = true
					break
				}
			}
			if !ok {
				pdTask.Lock()
				pdTask.errMsg = "下载 " + name + " 失败（所有镜像不可用）"
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

// uninstallWine 卸载 Wine
func (p *palDefenderAPI) uninstallWine(c *gin.Context) {
	if runtime.GOOS != "linux" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "仅支持 Linux"})
		return
	}
	cmd := exec.Command("apt-get", "remove", "-y", "wine64", "wine")
	setSysProcAttr(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("卸载失败: %v: %s", err, string(out))})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Wine 已卸载"})
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

// wineInstallState Wine 安装任务状态
type wineInstallState struct {
	running  bool
	done     bool
	success  bool
	log      strings.Builder
	errMsg   string
	percent  int
	message  string
	mu       sync.Mutex
}

var wineTask = &wineInstallState{}

func setWineProgress(pct int, msg string) {
	wineTask.mu.Lock()
	wineTask.percent = pct
	wineTask.message = msg
	wineTask.mu.Unlock()
}

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

		setWineProgress(10, "启用 i386 架构...")
		commands := []struct {
			args    []string
			percent int
			label   string
		}{
			{[]string{"dpkg", "--add-architecture", "i386"}, 10, "启用 i386 架构..."},
			{[]string{"apt-get", "update", "-y"}, 30, "更新软件源..."},
			{[]string{"apt-get", "install", "-y", "wine64"}, 50, "安装 Wine（可能需要几分钟）..."},
		}

		for _, step := range commands {
			setWineProgress(step.percent, step.label)
			wineTask.mu.Lock()
			wineTask.log.WriteString("$ " + strings.Join(step.args, " ") + "\n")
			wineTask.mu.Unlock()

			cmd := exec.Command(step.args[0], step.args[1:]...)
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
				wineTask.errMsg = fmt.Sprintf("执行 %s 失败: %v", step.args[0], err)
				wineTask.mu.Unlock()
				return
			}
		}

		setWineProgress(95, "验证安装...")
		if out, err := exec.Command("wine64", "--version").Output(); err == nil {
			wineTask.mu.Lock()
			wineTask.success = true
			wineTask.percent = 100
			wineTask.message = "安装完成"
			wineTask.log.WriteString("\n安装完成: " + strings.TrimSpace(string(out)) + "\n")
			wineTask.mu.Unlock()
		} else {
			wineTask.mu.Lock()
			wineTask.success = true
			wineTask.percent = 100
			wineTask.message = "Wine 安装完成"
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
		"percent": wineTask.percent,
		"message": wineTask.message,
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
		// Windows 版装在独立子目录 PalServer-Win
		win64 := filepath.Join(gameDir, "PalServer-Win", "Pal", "Binaries", "Win64")
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
