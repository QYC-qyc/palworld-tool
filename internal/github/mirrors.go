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
// 这些镜像对 github.com/release/download 大文件做反向代理。
func DownloadMirrors() []string {
	return []string{
		"https://ghfast.top/",
		"https://gh-proxy.com/",
		"https://ghproxy.net/",
		"https://mirror.ghproxy.com/",
		"https://ghproxy.cn/",
		"https://gh.h233.eu.org/",
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
	return DownloadToFileWithProgress(directURL, dst, nil)
}

// ProgressFunc 接收已下载字节数和总字节数（总大小未知时 total 为 0）。
type ProgressFunc func(downloaded, total int64)

// DownloadToFileWithProgress 经多个镜像尝试下载，并通过 onProgress 回报进度。
// 每个镜像先测速（Range 请求前 512KB），选最快的下载；失败自动切换。
func DownloadToFileWithProgress(directURL, dst string, onProgress ProgressFunc) error {
	// 先对各镜像做小范围测速，按响应速度排序
	ranked := rankMirrors(directURL)
	if len(ranked) == 0 {
		ranked = DownloadURLs(directURL)
	}

	var lastErr error
	for _, url := range ranked {
		if onProgress != nil {
			onProgress(0, 0)
		}
		if err := downloadOneWithProgress(url, dst, onProgress); err != nil {
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

// rankMirrors 对各镜像发起小范围测速（下载前 512KB），按响应速度降序排列。
func rankMirrors(directURL string) []string {
	type result struct {
		url   string
		speed float64 // MB/s
	}
	probeCli := &http.Client{Timeout: 8 * time.Second}
	var results []result
	for _, prefix := range DownloadMirrors() {
		url := prefix + directURL
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "PalAdmin")
		req.Header.Set("Range", "bytes=0-524287") // 前 512KB
		start := time.Now()
		resp, err := probeCli.Do(req)
		if err != nil {
			continue
		}
		n, _ := io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		elapsed := time.Since(start).Seconds()
		if (resp.StatusCode != 200 && resp.StatusCode != 206) || elapsed <= 0 || n == 0 {
			continue
		}
		results = append(results, result{url: url, speed: float64(n) / elapsed / 1024 / 1024})
	}
	// 按速度降序
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].speed > results[j-1].speed; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
	ranked := make([]string, 0, len(results)+1)
	hasDirect := false
	for _, r := range results {
		ranked = append(ranked, r.url)
		if strings.Contains(r.url, directURL) && !strings.Contains(r.url, "http") {
			hasDirect = true
		}
	}
	// 直连兜底
	if !hasDirect {
		ranked = append(ranked, directURL)
	}
	return ranked
}

func downloadOne(url, dst string) error {
	return downloadOneWithProgress(url, dst, nil)
}

func downloadOneWithProgress(url, dst string, onProgress ProgressFunc) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	var total int64
	if onProgress != nil {
		total = resp.ContentLength
	}
	var downloaded int64
	buf := make([]byte, 64*1024)
	var lastReport time.Time
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			downloaded += int64(n)
			if onProgress != nil && time.Since(lastReport) > 300*time.Millisecond {
				onProgress(downloaded, total)
				lastReport = time.Now()
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	if onProgress != nil {
		onProgress(downloaded, total)
	}
	return nil
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
