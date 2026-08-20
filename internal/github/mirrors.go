// Package github 提供统一的 GitHub 国内访问镜像配置与辅助函数。
// 所有从 GitHub 下载 release 资源、调用 api.github.com 的地方都应使用本包，
// 避免镜像列表散落各处导致不一致。
package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// DownloadMirrors 返回 GitHub release 文件下载（github.com/.../releases/download/...）的镜像前缀。
// 空字符串表示直连 GitHub。顺序即尝试顺序，国内镜像在前、直连兜底。
func DownloadMirrors() []string {
	return []string{
		"https://ghfast.top/",
		"https://gh-proxy.com/",
		"https://ghproxy.net/",
		"", // 直连
	}
}

// APIMirrors 返回 api.github.com 的镜像前缀。
// 镜像通过在完整 URL 前加代理前缀实现（如 https://ghproxy.net/https://api.github.com/...）。
func APIMirrors() []string {
	return []string{
		"", // 直连
		"https://ghproxy.net/https://",
		"https://ghfast.top/https://",
	}
}

// DownloadURLs 给定一个 GitHub 直链（如 https://github.com/owner/repo/releases/download/.../file），
// 返回经各镜像加速的完整 URL 列表（含直连）。
func DownloadURLs(directURL string) []string {
	urls := make([]string, 0, len(DownloadMirrors()))
	for _, prefix := range DownloadMirrors() {
		urls = append(urls, prefix+directURL)
	}
	return urls
}

// FetchRelease 经多个镜像尝试获取 GitHub 最新 release，解析到 out。
// apiURL 形如 https://api.github.com/repos/OWNER/REPO/releases/latest。
// 每个请求独立短超时，避免在不可达镜像上长时间卡住。
func FetchRelease(apiURL string, out interface{}) error {
	shortClient := &http.Client{Timeout: 20 * time.Second}
	var lastErr error
	for _, prefix := range APIMirrors() {
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

// DownloadToFile 经多个镜像尝试下载文件到 dst，返回 nil 表示成功。
// 会依次尝试 DownloadURLs(directURL)，任一成功即返回。
func DownloadToFile(directURL, dst string) error {
	var lastErr error
	for _, url := range DownloadURLs(directURL) {
		if err := downloadOne(url, dst); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr == nil {
		lastErr = errors.New("所有镜像均不可用")
	}
	return lastErr
}

func downloadOne(url, dst string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
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

// MirrorLabel 返回镜像前缀的可读名称（用于日志/进度显示）。
func MirrorLabel(prefix string) string {
	switch {
	case strings.Contains(prefix, "ghfast.top"):
		return "ghfast.top"
	case strings.Contains(prefix, "gh-proxy.com"):
		return "gh-proxy.com"
	case strings.Contains(prefix, "ghproxy.net"):
		return "ghproxy.net"
	case prefix == "":
		return "GitHub 直连"
	default:
		return strings.TrimSuffix(strings.TrimPrefix(prefix, "https://"), "/")
	}
}
