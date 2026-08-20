// Package webdata 负责把随程序分发的前端压缩包（web.dat）释放到磁盘，
// 避免向用户分发上万个小文件导致解压缓慢。
package webdata

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// EnsureExtracted 检查 dstDir 下是否已存在与 version 匹配的前端。
// 若不存在或版本不匹配，则从同目录下的 web.dat（zip 压缩包）解压到 dstDir。
// 返回最终前端目录路径（filepath.Join(dstDir, "web")）。
//
// webDatPath 是 web.dat 的完整路径；若为空，则在 exe 同目录查找。
func EnsureExtracted(dstDir, version, webDatPath string) (string, error) {
	webDir := filepath.Join(dstDir, "web")
	verFile := filepath.Join(webDir, ".version")

	// 已释放且版本一致，直接用
	if data, err := os.ReadFile(verFile); err == nil && strings.TrimSpace(string(data)) == version {
		if _, err := os.Stat(filepath.Join(webDir, "index.html")); err == nil {
			return webDir, nil
		}
	}

	// 确定 web.dat 位置
	if webDatPath == "" {
		exe, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("定位程序目录失败: %w", err)
		}
		webDatPath = filepath.Join(filepath.Dir(exe), "web.dat")
	}
	if _, err := os.Stat(webDatPath); err != nil {
		return "", fmt.Errorf("未找到前端资源包 %s：%w", webDatPath, err)
	}

	// 解压（先清旧目录）
	_ = os.RemoveAll(webDir)
	if err := unzip(webDatPath, webDir); err != nil {
		return "", fmt.Errorf("解压前端资源失败: %w", err)
	}

	// 写入版本标记
	_ = os.MkdirAll(webDir, 0o755)
	if err := os.WriteFile(verFile, []byte(version), 0o644); err != nil {
		return "", fmt.Errorf("写入版本标记失败: %w", err)
	}

	return webDir, nil
}

func unzip(zipPath, dst string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	for _, f := range r.File {
		// 防御 zip slip
		target := filepath.Join(dst, f.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("非法文件路径: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
