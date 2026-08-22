package gamesrv

import (
	"io"
	"os"
	"path/filepath"

	ghub "palworld-panel/internal/github"
)

// 本文件为 Manager 提供与部署模式无关的游戏文件系统访问。
// 所有读写均通过 Host 接口；业务层（api/service/tool）应使用这些方法，
// 不要直接对游戏目录用 os.*。

const dockerGameRoot = "/home/steam/palserver"

// IsDocker 报告当前是否通过 Docker 管控游戏服。
func (m *Manager) IsDocker() bool { return m.host != nil && m.host.Kind() == "docker" }

// GameRoot 返回游戏根目录（容器内或本地绝对路径）。
func (m *Manager) GameRoot() string {
	if m.host != nil {
		return m.host.GameRoot()
	}
	return m.winInstallDir()
}

// joinGame 拼接游戏根下的相对路径。
func (m *Manager) joinGame(elems ...string) string {
	return filepath.Join(append([]string{m.GameRoot()}, elems...)...)
}

// ---- 小文件读写（配置 ini、DLL、Token、Config.json 等）----

// ReadGameFile 读取游戏目录下的文件。
func (m *Manager) ReadGameFile(elems ...string) ([]byte, error) {
	rel := filepath.ToSlash(filepath.Join(elems...))
	if m.host != nil {
		return m.host.ReadFile(rel)
	}
	return os.ReadFile(m.joinGame(elems...))
}

// WriteGameFile 写入游戏目录下的文件，自动创建父目录。
func (m *Manager) WriteGameFile(data []byte, perm os.FileMode, elems ...string) error {
	rel := filepath.ToSlash(filepath.Join(elems...))
	if m.host != nil {
		return m.host.WriteFile(rel, data, perm)
	}
	abs := m.joinGame(elems...)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return err
	}
	return os.WriteFile(abs, data, perm)
}

// StatGameFile 返回游戏目录下文件信息。
func (m *Manager) StatGameFile(elems ...string) (os.FileInfo, error) {
	rel := filepath.ToSlash(filepath.Join(elems...))
	if m.host != nil {
		return m.host.Stat(rel)
	}
	return os.Stat(m.joinGame(elems...))
}

// MkdirAllGame 创建游戏目录。
func (m *Manager) MkdirAllGame(perm os.FileMode, elems ...string) error {
	rel := filepath.ToSlash(filepath.Join(elems...))
	if m.host != nil {
		return m.host.MkdirAll(rel, perm)
	}
	return os.MkdirAll(m.joinGame(elems...), perm)
}

// RemoveGameFile 删除游戏目录下的文件。
func (m *Manager) RemoveGameFile(elems ...string) error {
	rel := filepath.ToSlash(filepath.Join(elems...))
	if m.host != nil {
		return m.host.Remove(rel)
	}
	return os.Remove(m.joinGame(elems...))
}

// RemoveAllGame 递归删除游戏目录下的路径。
func (m *Manager) RemoveAllGame(elems ...string) error {
	rel := filepath.ToSlash(filepath.Join(elems...))
	if m.host != nil {
		return m.host.RemoveAll(rel)
	}
	return os.RemoveAll(m.joinGame(elems...))
}

// ListGameDir 列出游戏目录下的条目名称。
func (m *Manager) ListGameDir(elems ...string) ([]string, error) {
	rel := filepath.ToSlash(filepath.Join(elems...))
	if m.host != nil {
		return m.host.ListDir(rel)
	}
	entries, err := os.ReadDir(m.joinGame(elems...))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

// DownloadToGame 下载 URL 到游戏目录。
func (m *Manager) DownloadToGame(url string, perm os.FileMode, elems ...string) error {
	tmp, err := os.CreateTemp("", "paladm-dl-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	tmp.Close()
	if err := ghub.DownloadToFile(url, tmpPath); err != nil {
		return err
	}
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return err
	}
	return m.WriteGameFile(data, perm, elems...)
}

// ---- 存档整树传输（备份/回档）----

const savedRel = "Pal/Saved"

// FetchSavedToLocal 把游戏的 Pal/Saved 复制到 localRoot/Pal/Saved。
func (m *Manager) FetchSavedToLocal(localRoot string) error {
	if m.host != nil {
		return m.host.FetchSaved(localRoot)
	}
	palDir := filepath.Join(localRoot, "Pal")
	if err := os.MkdirAll(palDir, 0755); err != nil {
		return err
	}
	return copyTree(m.joinGame("Pal", "Saved"), filepath.Join(localRoot, "Pal", "Saved"))
}

// PushLocalToSaved 把 localRoot/Pal/Saved 推回游戏目录覆盖。
func (m *Manager) PushLocalToSaved(localRoot string) error {
	if m.host != nil {
		return m.host.PushSaved(localRoot)
	}
	localSaved := filepath.Join(localRoot, "Pal", "Saved")
	target := m.joinGame("Pal", "Saved")
	_ = os.RemoveAll(target)
	return copyTree(localSaved, target)
}

// copyTree 递归拷贝目录（裸机部署的存档回退用）。
func copyTree(src, dst string) error {
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
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}
