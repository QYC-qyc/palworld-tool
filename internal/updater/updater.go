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

type ReleaseInfo struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	Body        string  `json:"body"`
	PublishedAt string  `json:"published_at"`
	Assets      []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type Progress struct {
	Stage   string  `json:"stage"`
	Message string  `json:"message"`
	Percent float64 `json:"percent"`
	Version string  `json:"version"`
}

var (
	httpClient     = &http.Client{Timeout: 0}
	currentVersion = "dev"
)

// 下载镜像（和 install.sh 一致）
var dlMirrors = []string{
	"https://ghfast.top/",
	"https://gh-proxy.com/",
	"https://ghproxy.net/",
	"",
}

func SetVersion(v string) {
	if v != "" {
		currentVersion = v
	}
}

func CurrentVersion() string {
	return currentVersion
}

func Check() (*ReleaseInfo, bool, error) {
	cli := &http.Client{Timeout: 8 * time.Second}
	endpoints := []string{
		repoAPI,
		"https://ghproxy.net/https://api.github.com/repos/QYC-qyc/palworld-tool/releases/latest",
		"https://ghfast.top/https://api.github.com/repos/QYC-qyc/palworld-tool/releases/latest",
	}
	for _, apiURL := range endpoints {
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
	return nil, false, fmt.Errorf("无法连接更新服务器")
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

// DoUpdate 后端直接下载二进制并替换，通过 onProgress 实时回调进度。
// 调用者应在独立 goroutine 中调用此函数。
func DoUpdate(rel *ReleaseInfo, installDir, service string, onProgress func(Progress)) error {
	if onProgress == nil {
		onProgress = func(Progress) {}
	}
	if runtime.GOOS != "linux" {
		return fmt.Errorf("在线更新仅支持 Linux")
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
		// 尝试旧命名
		for _, a := range rel.Assets {
			if strings.Contains(a.Name, runtime.GOARCH) && strings.HasSuffix(a.Name, ".tar.gz") {
				downloadURL = a.BrowserDownloadURL
				asset = a.Name
				break
			}
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("最新版本未找到 %s", asset)
	}

	tmpTar := filepath.Join(os.TempDir(), asset)
	defer os.Remove(tmpTar)

	// 逐镜像下载，带进度
	var totalSize int64
	for _, a := range rel.Assets {
		if a.BrowserDownloadURL == downloadURL {
			totalSize = a.Size
		}
	}

	var lastErr error
	for i, prefix := range dlMirrors {
		url := prefix + downloadURL
		onProgress(Progress{Stage: "download", Message: fmt.Sprintf("镜像 %d/%d 下载中...", i+1, len(dlMirrors)), Percent: 0, Version: rel.TagName})
		if err := downloadWithProgress(url, tmpTar, totalSize, func(pct float64, speed string) {
			onProgress(Progress{Stage: "download", Message: "下载中 " + speed, Percent: pct, Version: rel.TagName})
		}); err != nil {
			lastErr = err
			onProgress(Progress{Stage: "download", Message: "该镜像失败，切换中..."})
			continue
		}
		lastErr = nil
		break
	}
	if lastErr != nil {
		return fmt.Errorf("所有镜像下载失败: %w", lastErr)
	}

	onProgress(Progress{Stage: "extract", Message: "解压文件...", Percent: 92})
	// 先解压到临时目录，避免覆盖正在运行的二进制
	tmpDir := filepath.Join(os.TempDir(), "paladmin-update-"+time.Now().Format("20060102150405"))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := extractTarGz(tmpTar, tmpDir); err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}

	// 设置权限
	_ = os.Chmod(filepath.Join(tmpDir, "paladmin"), 0755)
	if sav := filepath.Join(tmpDir, "sav_cli"); fileExists(sav) {
		_ = os.Chmod(sav, 0755)
	}

	onProgress(Progress{Stage: "restart", Message: "正在替换文件并重启...", Percent: 97})
	// 用脚本完成原子替换+重启，脱离当前进程
	replaceScript := fmt.Sprintf(`sleep 1
cp -rf %s/* %s/
systemctl restart %s
`, tmpDir, installDir, service)
	cmd := exec.Command("setsid", "bash", "-c", replaceScript)
	cmd.SysProcAttr = newSysProcAttr()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动替换脚本失败: %w", err)
	}

	return nil
}

func downloadWithProgress(url, dst string, totalSize int64, onProgress func(float64, string)) error {
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

	start := time.Now()
	buf := make([]byte, 64*1024)
	var downloaded int64
	var lastPct int
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			downloaded += int64(n)
			if totalSize > 0 {
				pct := int(float64(downloaded) / float64(totalSize) * 100)
				if pct != lastPct {
					lastPct = pct
					elapsed := time.Since(start).Seconds()
					speed := ""
					if elapsed > 0 {
						speed = fmt.Sprintf("%.1f MB/s", float64(downloaded)/elapsed/1024/1024)
					}
					onProgress(float64(pct), speed)
				}
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
		name := filepath.Clean(hdr.Name)
		if strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			continue
		}
		target := filepath.Join(dst, name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(target, os.FileMode(hdr.Mode))
		case tar.TypeReg:
			top := strings.SplitN(name, string(os.PathSeparator), 2)[0]
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
