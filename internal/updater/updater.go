// Package updater 实现面板二进制与前端资源的在线自更新。
package updater

import (
	"archive/tar"
	"compress/gzip"
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
	TagName    string  `json:"tag_name"`
	Name       string  `json:"name"`
	Body       string  `json:"body"`
	PublishedAt string `json:"published_at"`
	Assets     []Asset `json:"assets"`
}

// Asset 附件
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

// CurrentVersion 当前版本（由 main 通过 SetVersion 注入）
var currentVersion = "dev"

func SetVersion(v string) {
	if v != "" {
		currentVersion = v
	}
}

// CurrentVersion 返回当前版本号
func CurrentVersion() string {
	return currentVersion
}

// Check 检查最新版本
func Check() (*ReleaseInfo, bool, error) {
	req, err := http.NewRequest("GET", repoAPI, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("检查更新失败（网络不通）: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("检查更新失败: HTTP %d", resp.StatusCode)
	}
	var rel ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, false, err
	}
	hasUpdate := normalizeVersion(rel.TagName) != normalizeVersion(currentVersion) &&
		normalizeVersion(rel.TagName) != ""
	return &rel, hasUpdate, nil
}

func normalizeVersion(v string) string {
	v = strings.TrimPrefix(v, "v")
	return v
}

// assetName 根据架构返回对应附件名
func assetName() string {
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		return "paladmin_linux_amd64.tar.gz"
	case "arm64":
		return "paladmin_linux_arm64.tar.gz"
	default:
		return ""
	}
}

// DoUpdate 执行更新：下载→替换→重启。
// installDir 为面板安装目录（如 /opt/paladmin），service 为 systemd 服务名。
// 进度通过 logf 回调输出。
func DoUpdate(rel *ReleaseInfo, installDir, service string, logf func(string)) error {
	asset := assetName()
	if asset == "" {
		return fmt.Errorf("不支持的架构: %s", runtime.GOARCH)
	}
	var downloadURL string
	for _, a := range rel.Assets {
		if a.Name == asset {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("最新版本未找到 %s", asset)
	}

	logf(fmt.Sprintf("开始下载 %s ...", asset))
	tmpTar := filepath.Join(os.TempDir(), asset)
	if err := downloadFile(downloadURL, tmpTar); err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer os.Remove(tmpTar)
	logf("下载完成，开始解压替换...")

	if err := extractTarGz(tmpTar, installDir, logf); err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}

	// 设置可执行权限
	binPath := filepath.Join(installDir, "paladmin")
	_ = os.Chmod(binPath, 0755)
	if sav := filepath.Join(installDir, "sav_cli"); fileExists(sav) {
		_ = os.Chmod(sav, 0755)
	}

	logf("文件替换完成，正在重启服务...")
	go func() {
		time.Sleep(2 * time.Second)
		cmd := exec.Command("systemctl", "restart", service)
		cmd.SysProcAttr = newSysProcAttr()
		_ = cmd.Run()
	}()
	return nil
}

func downloadFile(url, dst string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := httpClient.Do(req)
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

// extractTarGz 解压 tar.gz 到目标目录，覆盖同名文件/目录
func extractTarGz(src, dst string, logf func(string)) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	cleaned := map[string]bool{
		"web":  false,
		"data": false,
	}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// 安全：禁止绝对路径和 ..
		cleanName := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
			continue
		}
		target := filepath.Join(dst, cleanName)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			// web 和 data 目录先整体清空再写入，避免旧 hash 文件残留
			top := strings.SplitN(cleanName, string(os.PathSeparator), 2)[0]
			if !cleaned[top] && (top == "web" || top == "data") {
				_ = os.RemoveAll(filepath.Join(dst, top))
				cleaned[top] = true
				logf(fmt.Sprintf("已清理旧 %s/ 目录", top))
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
