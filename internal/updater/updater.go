// Package updater 实现面板二进制与前端资源的在线自更新。
package updater

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
)

const (
	repoAPI   = "https://api.github.com/repos/QYC-qyc/palworld-tool/releases/latest"
	userAgent = "PalAdmin-Updater"
)

// ReleaseInfo GitHub release 信息
type ReleaseInfo struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	Body        string  `json:"body"`
	PublishedAt string  `json:"published_at"`
	Assets      []Asset `json:"assets"`
}

// Asset 附件
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// Progress 进度回调
type Progress struct {
	Stage    string  `json:"stage"`    // check/download/extract/restart/done/error
	Message  string  `json:"message"`  // 人类可读信息
	Percent  float64 `json:"percent"`  // 下载进度 0-100
	Version  string  `json:"version"`  // 目标版本
}

var httpClient = &http.Client{Timeout: 60 * time.Second}

var currentVersion = "dev"

func SetVersion(v string) {
	if v != "" {
		currentVersion = v
	}
}

func CurrentVersion() string {
	return currentVersion
}

// Check 检查最新版本，依次尝试直连和镜像
func Check() (*ReleaseInfo, bool, error) {
	cli := &http.Client{Timeout: 8 * time.Second}

	// 尝试多个 GitHub API 镜像源
	apiEndpoints := []string{
		repoAPI,
		"https://ghproxy.net/https://api.github.com/repos/QYC-qyc/palworld-tool/releases/latest",
		"https://ghfast.top/https://api.github.com/repos/QYC-qyc/palworld-tool/releases/latest",
	}

	for _, apiURL := range apiEndpoints {
		req, _ := http.NewRequest("GET", apiURL, nil)
		req.Header.Set("User-Agent", userAgent)
		resp, err := cli.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}
		var rel ReleaseInfo
		err = json.NewDecoder(resp.Body).Decode(&rel)
		resp.Body.Close()
		if err == nil && rel.TagName != "" && len(rel.Assets) > 0 {
			hasUpdate := normalizeVersion(rel.TagName) != normalizeVersion(currentVersion)
			return &rel, hasUpdate, nil
		}
	}

	return nil, false, fmt.Errorf("无法连接更新服务器（GitHub 访问受限），可手动执行安装脚本更新")
}

func normalizeVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

const installScriptURL = "https://gitee.com/QYC-qyc/palworld-tool/raw/main/scripts/install.sh"

// DoUpdate 直接执行官方安装脚本更新（和手动 curl|bash 效果一致）
func DoUpdate(rel *ReleaseInfo, installDir, service string, onProgress func(Progress)) error {
	if onProgress == nil {
		onProgress = func(Progress) {}
	}

	if runtime.GOOS == "windows" {
		return fmt.Errorf("在线更新仅支持 Linux")
	}

	onProgress(Progress{Stage: "download", Message: "正在下载安装脚本...", Percent: 10, Version: rel.TagName})

	// 下载脚本到临时文件
	tmpScript := filepath.Join(os.TempDir(), "paladmin-install.sh")
	resp, err := httpClient.Get(installScriptURL)
	if err != nil {
		return fmt.Errorf("下载脚本失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("下载脚本失败: HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(tmpScript)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	f.Close()
	defer os.Remove(tmpScript)
	_ = os.Chmod(tmpScript, 0755)

	onProgress(Progress{Stage: "download", Message: "正在下载并安装最新版本...", Percent: 30, Version: rel.TagName})

	// 用 nohup + setsid 让脚本完全脱离当前进程独立运行。
	// 脚本内部会 systemctl stop paladmin → 替换文件 → systemctl start paladmin。
	// 当前进程不能自杀，否则脚本还没下载完二进制就被 systemd 重启为旧版本。
	cmd := exec.Command("setsid", "bash", "-c",
		fmt.Sprintf("nohup bash %s > /tmp/paladmin-update.log 2>&1 &", tmpScript))
	cmd.SysProcAttr = newSysProcAttr()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动安装脚本失败: %w", err)
	}

	onProgress(Progress{Stage: "restart", Message: "更新已在后台启动，服务将在下载完成后重启...", Percent: 90, Version: rel.TagName})
	return nil
}
