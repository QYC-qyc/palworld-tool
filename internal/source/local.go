package source

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// FindLevelSav 在 Saved 目录中查找 Level.sav 文件（路径含动态 GUID）
func FindLevelSav(savedDir string) (string, error) {
	var levelPath string
	err := filepath.Walk(savedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.EqualFold(info.Name(), "Level.sav") {
			levelPath = path
			return io.EOF
		}
		return nil
	})
	if err != nil && err != io.EOF {
		return "", err
	}
	if levelPath == "" {
		return "", fmt.Errorf("在 %s 中未找到 Level.sav", savedDir)
	}
	return levelPath, nil
}

// CopyFromLocal 将存档目录（Level.sav 及 Players/）复制到临时目录，返回其中 Level.sav 路径
func CopyFromLocal(savedDir, way string) (string, error) {
	levelSrc, err := FindLevelSav(savedDir)
	if err != nil {
		return "", err
	}
	tmpDir := filepath.Join(os.TempDir(), "paladm-"+way+"-"+uuid.NewString()[:8])
	saveRoot := filepath.Dir(levelSrc)

	err = filepath.Walk(saveRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(saveRoot, path)
		if err != nil {
			return err
		}
		target := filepath.Join(tmpDir, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		// 只复制 Level.sav 和 Players 目录
		if !strings.EqualFold(info.Name(), "Level.sav") && !inPlayersDir(rel) {
			return nil
		}
		return copyFile(path, target)
	})
	if err != nil {
		return "", err
	}
	return filepath.Join(tmpDir, "Level.sav"), nil
}

func inPlayersDir(rel string) bool {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, p := range parts {
		if p == "Players" {
			return true
		}
	}
	return false
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// CopySavedTree 递归拷贝整个 Saved 目录（本地模式备份用）
func CopySavedTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}
