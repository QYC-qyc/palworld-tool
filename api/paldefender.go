package api

import (
	"encoding/json"
	"errors"
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
	"paladmin/service"
)

var httpClient = &http.Client{Timeout: 5 * time.Minute}

const (
	palDefenderVersionAPI = "https://api.github.com/repos/Ultimeit/PalDefender/releases/latest"
	palDefenderDLL1       = "d3d9.dll"
	palDefenderDLL2       = "PalDefender.dll"

	protonGeReleasesAPI = "https://api.github.com/repos/GloriousEggroll/proton-ge-custom/releases/latest"
	protonInstallDir    = "/opt/GE-Proton"
)

type palDefenderAPI struct{}

type pdStatus struct {
	Installed     bool   `json:"installed"`
	Win64Path     string `json:"win64_path"`
	ProtonPresent bool   `json:"proton_present"`
	ProtonPath    string `json:"proton_path"`
	ProtonVersion string `json:"proton_version"`
	WineInstalled bool   `json:"wine_installed"`
	D3d9Exists    bool   `json:"d3d9_exists"`
	PdExists      bool   `json:"pd_exists"`
	GameDir       string `json:"game_dir"`
}

// status 检测 PalDefender 安装状态和 Proton
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

	if runtime.GOOS != "linux" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "仅支持 Linux"})
		return
	}

	st := p.detectAt(req.GameDir)
	if !st.ProtonPresent {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "未检测到 Proton，请先一键安装 Proton"})
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
		var rel struct {
			TagName string `json:"tag_name"`
			Assets  []struct {
				Name               string `json:"name"`
				BrowserDownloadURL string `json:"browser_download_url"`
			} `json:"assets"`
		}
		if err := fetchGitHubRelease(palDefenderVersionAPI, &rel); err != nil {
			pdTask.Lock()
			pdTask.errMsg = "检查版本失败: " + err.Error()
			pdTask.Unlock()
			return
		}

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

// uninstallWine 卸载 Wine（Proton 不再依赖 Wine，但保留此接口供用户清理旧 Wine）
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

// protonInstallState Proton 安装任务状态
type protonInstallState struct {
	running bool
	done    bool
	success bool
	log     strings.Builder
	errMsg  string
	percent int
	message string
	mu      sync.Mutex
}

var protonTask = &protonInstallState{}

func setProtonProgress(pct int, msg string) {
	protonTask.mu.Lock()
	protonTask.percent = pct
	protonTask.message = msg
	protonTask.mu.Unlock()
}

// detectOS 读取 /etc/os-release，返回 ID 和 VERSION_ID。
func detectOS() (id, version string) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ID=") {
			id = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		}
		if strings.HasPrefix(line, "VERSION_ID=") {
			version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
		}
	}
	return id, version
}

// installProton 后台执行 GE-Proton 安装，立即返回
func (p *palDefenderAPI) installProton(c *gin.Context) {
	if runtime.GOOS != "linux" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "仅支持 Linux"})
		return
	}

	protonTask.mu.Lock()
	if protonTask.running {
		protonTask.mu.Unlock()
		c.JSON(http.StatusOK, gin.H{"running": true, "message": "正在安装中..."})
		return
	}
	protonTask.running = true
	protonTask.done = false
	protonTask.success = false
	protonTask.errMsg = ""
	protonTask.log.Reset()
	protonTask.mu.Unlock()

	go func() {
		defer func() {
			protonTask.mu.Lock()
			protonTask.running = false
			protonTask.done = true
			protonTask.mu.Unlock()
		}()

		logWrite := func(s string) {
			protonTask.mu.Lock()
			protonTask.log.WriteString(s)
			protonTask.mu.Unlock()
		}

		// 1. 检测操作系统（5%）
		setProtonProgress(5, "检测操作系统...")
		osID, _ := detectOS()
		osID = strings.ToLower(osID)
		isDebian := osID == "ubuntu" || osID == "debian"
		isArch := osID == "arch"
		if !isDebian && !isArch {
			protonTask.mu.Lock()
			protonTask.errMsg = "暂不支持自动安装 Proton，请手动安装 GE-Proton 并在设置中指定路径"
			protonTask.mu.Unlock()
			return
		}
		logWrite(fmt.Sprintf("检测到系统: %s\n", osID))

		// 2. 安装依赖（40%）
		setProtonProgress(40, "安装系统依赖...")
		// apt 非交互环境变量，避免弹窗/交互导致卡住
		aptEnv := append(os.Environ(),
			"DEBIAN_FRONTEND=noninteractive",
			"NEEDRESTART_MODE=a",
			"APT_LISTCHANGES_FRONTEND=none",
		)
		var depCmds [][]string
		if isDebian {
			// Debian/Ubuntu 下 GE-Proton 运行所需的 32/64 位库。
			// 注意：steam-libs-i386 是 Arch 包名，Debian 系不存在，勿用。
			depCmds = [][]string{
				{"dpkg", "--add-architecture", "i386"},
				{"apt-get", "update", "-y"},
				{"apt-get", "install", "-y", "--no-install-recommends",
					"curl", "tar", "xz-utils", "ca-certificates",
					"lib32gcc-s1", "lib32stdc++6",
					"libc6:i386", "libstdc++6:i386", "libgcc-s1:i386"},
			}
		} else {
			depCmds = [][]string{
				{"pacman", "-Sy", "--noconfirm", "--needed",
					"curl", "tar", "xz", "wine", "lib32-gcc-libs", "lib32-glibc"},
			}
		}
		for _, args := range depCmds {
			logWrite("$ " + strings.Join(args, " ") + "\n")
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Env = aptEnv
			setSysProcAttr(cmd)
			stdout, _ := cmd.StdoutPipe()
			cmd.Stderr = cmd.Stdout
			if err := cmd.Start(); err != nil {
				protonTask.mu.Lock()
				protonTask.errMsg = "启动失败: " + err.Error()
				protonTask.mu.Unlock()
				return
			}
			buf := make([]byte, 4096)
			for {
				n, err := stdout.Read(buf)
				if n > 0 {
					logWrite(string(buf[:n]))
				}
				if err != nil {
					break
				}
			}
			if err := cmd.Wait(); err != nil {
				protonTask.mu.Lock()
				protonTask.errMsg = fmt.Sprintf("执行 %s 失败（exit %v）：可能是软件源问题或包名不匹配，请查看上方日志", strings.Join(args, " "), err)
				protonTask.mu.Unlock()
				return
			}
		}
		// Debian 系安装后尝试修复可能的破损依赖（非致命）
		if isDebian {
			fixCmd := exec.Command("apt-get", "install", "-y", "-f")
			fixCmd.Env = aptEnv
			_ = fixCmd.Run()
		}

		// 3. 下载最新 GE-Proton（70%）—— 经镜像获取版本，避免直连 GitHub API 卡住
		setProtonProgress(70, "获取 GE-Proton 最新版本...")
		var rel struct {
			TagName string `json:"tag_name"`
			Assets  []struct {
				Name               string `json:"name"`
				BrowserDownloadURL string `json:"browser_download_url"`
			} `json:"assets"`
		}
		if err := fetchGitHubRelease(protonGeReleasesAPI, &rel); err != nil {
			protonTask.mu.Lock()
			protonTask.errMsg = "获取 GE-Proton 版本失败: " + err.Error()
			protonTask.mu.Unlock()
			return
		}

		var tarballURL string
		for _, a := range rel.Assets {
			if strings.HasSuffix(a.Name, ".tar.gz") {
				tarballURL = a.BrowserDownloadURL
				break
			}
		}
		if tarballURL == "" {
			protonTask.mu.Lock()
			protonTask.errMsg = "未找到 GE-Proton 压缩包"
			protonTask.mu.Unlock()
			return
		}
		logWrite(fmt.Sprintf("下载 GE-Proton %s: %s\n", rel.TagName, tarballURL))

		// 下载到临时文件
		tmpFile, err := os.CreateTemp("", "ge-proton-*.tar.gz")
		if err != nil {
			protonTask.mu.Lock()
			protonTask.errMsg = "创建临时文件失败: " + err.Error()
			protonTask.mu.Unlock()
			return
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		dlMirrors := []string{
			"https://ghfast.top/",
			"https://gh-proxy.com/",
			"https://ghproxy.net/",
			"",
		}
		var dlOk bool
		for _, prefix := range dlMirrors {
			if err := downloadFile(prefix+tarballURL, tmpPath); err == nil {
				dlOk = true
				break
			}
		}
		tmpFile.Close()
		if !dlOk {
			protonTask.mu.Lock()
			protonTask.errMsg = "下载 GE-Proton 失败（所有镜像不可用）"
			protonTask.mu.Unlock()
			return
		}

		// 4. 解压（90%）
		setProtonProgress(90, "解压 GE-Proton...")
		if err := os.MkdirAll(protonInstallDir, 0755); err != nil {
			protonTask.mu.Lock()
			protonTask.errMsg = "创建安装目录失败: " + err.Error()
			protonTask.mu.Unlock()
			return
		}
		// 先清空旧版本
		entries, _ := os.ReadDir(protonInstallDir)
		for _, e := range entries {
			_ = os.RemoveAll(filepath.Join(protonInstallDir, e.Name()))
		}
		tarCmd := exec.Command("tar", "-xzf", tmpPath, "-C", protonInstallDir, "--strip-components=1")
		setSysProcAttr(tarCmd)
		if out, err := tarCmd.CombinedOutput(); err != nil {
			protonTask.mu.Lock()
			protonTask.errMsg = fmt.Sprintf("解压失败: %v: %s", err, string(out))
			protonTask.mu.Unlock()
			return
		}
		logWrite(fmt.Sprintf("已解压到 %s\n", protonInstallDir))

		// 5. 验证（100%）
		setProtonProgress(100, "验证安装...")
		protonExe := filepath.Join(protonInstallDir, "proton")
		if info, err := os.Stat(protonExe); err != nil || info.IsDir() {
			protonTask.mu.Lock()
			protonTask.errMsg = "验证失败：未找到 proton 可执行文件"
			protonTask.mu.Unlock()
			return
		}
		_ = os.Chmod(protonExe, 0755)
		// 记录版本
		versionFile := filepath.Join(protonInstallDir, "version")
		versionStr := rel.TagName
		if data, err := os.ReadFile(versionFile); err == nil {
			versionStr = strings.TrimSpace(string(data))
		}
		protonTask.mu.Lock()
		protonTask.success = true
		protonTask.percent = 100
		protonTask.message = fmt.Sprintf("GE-Proton %s 安装完成", versionStr)
		protonTask.log.WriteString("\n安装完成: " + versionStr + "\n")
		protonTask.mu.Unlock()
	}()

	c.JSON(http.StatusOK, gin.H{"running": true, "message": "开始安装 GE-Proton..."})
}

// protonInstallStatus 返回 Proton 安装进度
func (p *palDefenderAPI) protonInstallStatus(c *gin.Context) {
	protonTask.mu.Lock()
	defer protonTask.mu.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"running": protonTask.running,
		"done":    protonTask.done,
		"success": protonTask.success,
		"error":   protonTask.errMsg,
		"percent": protonTask.percent,
		"message": protonTask.message,
		"log":     protonTask.log.String(),
	})
}

// uninstallProton 卸载 GE-Proton（仅删除安装目录，不卸载系统依赖）
func (p *palDefenderAPI) uninstallProton(c *gin.Context) {
	if runtime.GOOS != "linux" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "仅支持 Linux"})
		return
	}
	if err := os.RemoveAll(protonInstallDir); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "删除失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "GE-Proton 已卸载"})
}

// detectProtonExe 检测 Proton 可执行文件路径。
// 优先使用设置 proton.path；为空时按顺序查找常见安装位置。
func detectProtonExe() string {
	if db != nil {
		if p := service.GetSetting(db, service.SettingProtonPath); p != "" {
			if info, err := os.Stat(p); err == nil && !info.IsDir() {
				return p
			}
		}
	}
	candidates := []string{
		filepath.Join(protonInstallDir, "proton"),
		"/usr/bin/proton",
		"/usr/local/bin/proton",
	}
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	patterns := []string{
		"/home/*/.steam/steam/compatibilitytools.d/GE-Proton*/proton",
		"/root/.steam/steam/compatibilitytools.d/GE-Proton*/proton",
	}
	for _, pat := range patterns {
		matches, err := filepath.Glob(pat)
		if err != nil || len(matches) == 0 {
			continue
		}
		return matches[len(matches)-1]
	}
	return ""
}

func (p *palDefenderAPI) detect() pdStatus {
	return p.detectAt("")
}

func (p *palDefenderAPI) detectAt(gameDir string) pdStatus {
	st := pdStatus{GameDir: gameDir}

	// 检测 Proton
	if protonPath := detectProtonExe(); protonPath != "" {
		st.ProtonPresent = true
		st.ProtonPath = protonPath
		// 读取版本：优先 GE-Proton/version 文件，否则尝试 proton --version
		versionFile := filepath.Join(filepath.Dir(protonPath), "version")
		if data, err := os.ReadFile(versionFile); err == nil {
			st.ProtonVersion = strings.TrimSpace(string(data))
		} else if out, err := exec.Command(protonPath, "--version").Output(); err == nil {
			st.ProtonVersion = strings.TrimSpace(string(out))
		}
	}

	// 检测 Wine 是否已安装（供前端显示"卸载 Wine"按钮）
	if _, err := exec.LookPath("wine64"); err == nil {
		st.WineInstalled = true
	} else if _, err := exec.LookPath("wine"); err == nil {
		st.WineInstalled = true
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

// githubAPIMirrors 返回 GitHub API 的镜像前缀（含直连）。
// 国内直连 api.github.com 常超时，通过这些代理加速。
var githubAPIMirrors = []string{
	"", // 直连
	"https://ghproxy.net/https://",
	"https://ghfast.top/https://",
}

// fetchGitHubRelease 经多个镜像尝试获取 GitHub 最新 release，解析到 out。
// 每个请求用独立短超时，避免在不可达的镜像上长时间卡住。
func fetchGitHubRelease(apiURL string, out interface{}) error {
	shortClient := &http.Client{Timeout: 20 * time.Second}
	var lastErr error
	for _, prefix := range githubAPIMirrors {
		url := prefix + apiURL
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "PalAdmin")
		resp, err := shortClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		}
		err = json.NewDecoder(resp.Body).Decode(out)
		resp.Body.Close()
		if err == nil {
			return nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("所有镜像均不可用")
	}
	return fmt.Errorf("获取 release 失败: %w", lastErr)
}
