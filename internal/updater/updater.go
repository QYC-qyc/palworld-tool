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

var httpClient = &http.Client{Timeout: 0} // 不设总超时，由流式下载控制

var currentVersion = "dev"

func SetVersion(v string) {
	if v != "" {
		currentVersion = v
	}
}

func CurrentVersion() string {
	return currentVersion
}

// 镜像列表（和 install.sh 保持一致），最后直连 GitHub
var mirrors = []string{
	"https://ghfast.top/https://github.com",
	"https://gh-proxy.com/https://github.com",
	"https://ghproxy.net/https://github.com",
	"https://github.com",
}

// Check 检查最新版本
func Check() (*ReleaseInfo, bool, error) {
	req, _ := http.NewRequest("GET", repoAPI, nil)
	req.Header.Set("User-Agent", userAgent)
	cli := &http.Client{Timeout: 15 * time.Second}
	resp, err := cli.Do(req)
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
	return strings.TrimPrefix(v, "v")
}

func assetName() string {
	switch runtime.GOARCH {
	case "amd64":
		return "paladmin_linux_amd64.tar.gz"
	case "arm64":
		return "paladmin_linux_arm64.tar.gz"
	default:
		return ""
	}
}

// DoUpdate 执行更新，通过 onProgress 实时回调进度
func DoUpdate(rel *ReleaseInfo, installDir, service string, onProgress func(Progress)) error {
	if onProgress == nil {
		onProgress = func(Progress) {}
	}

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

	onProgress(Progress{Stage: "download", Message: "准备下载 " + asset, Version: rel.TagName})

	tmpTar := filepath.Join(os.TempDir(), asset)
	defer os.Remove(tmpTar)

	// 逐镜像尝试下载
	var lastErr error
	for i, base := range mirrors {
		url := downloadURL
		if base != "https://github.com" {
			url = base + strings.TrimPrefix(downloadURL, "https://github.com")
		}
		onProgress(Progress{Stage: "download", Message: fmt.Sprintf("镜像 %d/%d: %s", i+1, len(mirrors), base), Percent: 0})
		if err := downloadFileWithProgress(url, tmpTar, rel.SizeFor(asset), func(pct float64, speed string) {
			onProgress(Progress{
				Stage: "download", Message: fmt.Sprintf("下载中 %s", speed),
				Percent: pct, Version: rel.TagName,
			})
		}); err != nil {
			lastErr = err
			onProgress(Progress{Stage: "download", Message: "该镜像失败: " + err.Error()})
			continue
		}
		lastErr = nil
		break
	}
	if lastErr != nil {
		onProgress(Progress{Stage: "error", Message: "所有镜像下载失败"})
		return fmt.Errorf("下载失败: %w", lastErr)
	}

	onProgress(Progress{Stage: "extract", Message: "解压并替换文件...", Percent: 100})
	if err := extractTarGz(tmpTar, installDir); err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}

	_ = os.Chmod(filepath.Join(installDir, "paladmin"), 0755)
	if sav := filepath.Join(installDir, "sav_cli"); fileExists(sav) {
		_ = os.Chmod(sav, 0755)
	}

	onProgress(Progress{Stage: "restart", Message: "正在重启服务..."})
	go func() {
		time.Sleep(1500 * time.Millisecond)
		cmd := exec.Command("systemctl", "restart", service)
		cmd.SysProcAttr = newSysProcAttr()
		out, err := cmd.CombinedOutput()
		if err != nil {
			// 非 systemd 环境（如 Docker），直接退出让容器重启
			os.Exit(0)
		}
		_ = out
	}()
	return nil
}

func (r *ReleaseInfo) SizeFor(asset string) int64 {
	for _, a := range r.Assets {
		if a.Name == asset {
			return a.Size
		}
	}
	return 0
}

func downloadFileWithProgress(url, dst string, totalSize int64, onProgress func(float64, string)) error {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", userAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > 0 {
		totalSize = resp.ContentLength
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	startTime := time.Now()
	buf := make([]byte, 32*1024)
	var downloaded int64
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			downloaded += int64(n)
			if totalSize > 0 {
				pct := float64(downloaded) / float64(totalSize) * 100
				elapsed := time.Since(startTime).Seconds()
				speed := ""
				if elapsed > 0 {
					bytesPerSec := float64(downloaded) / elapsed
					speed = fmt.Sprintf("(%.1f MB/s)", bytesPerSec/1024/1024)
				}
				onProgress(pct, speed)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTarGz(src, dst string) error {
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
	cleaned := map[string]bool{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		cleanName := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
			continue
		}
		target := filepath.Join(dst, cleanName)
		switch hdr.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(target, os.FileMode(hdr.Mode))
		case tar.TypeReg:
			top := strings.SplitN(cleanName, string(os.PathSeparator), 2)[0]
			if !cleaned[top] && (top == "web" || top == "data") {
				_ = os.RemoveAll(filepath.Join(dst, top))
				cleaned[top] = true
			}
			_ = os.MkdirAll(filepath.Dir(target), 0755)
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
