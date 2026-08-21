package gamesrv

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ghub "paladmin/internal/github"
)

// 本文件为 Manager 提供与部署模式无关的"游戏文件系统"访问：
//   - 本地模式（m.docker == nil）：直接用 os.* 操作 winInstallDir()
//   - Docker 模式（m.docker != nil）：通过 docker exec / tar 流操作容器内 /home/steam/palserver
// 业务层（api/service/tool）不应再直接对游戏目录用 os.*，否则容器内会写错位置。

// dockerGameRoot 容器内游戏安装根目录（steamcmd force_install_dir 指向这里）
const dockerGameRoot = "/home/steam/palserver"

// IsDocker 报告当前是否通过 Docker 管控游戏服。
func (m *Manager) IsDocker() bool { return m.docker != nil }

// gameRoot 返回游戏根目录（含 Pal/ 的目录）。
// Docker 下为容器内路径；本地为 Windows 版独立安装目录。
func (m *Manager) gameRoot() string {
	if m.docker != nil {
		return dockerGameRoot
	}
	return m.winInstallDir()
}

// joinGame 拼接游戏根下的相对路径（按段传入，跨平台自动处理分隔符）。
func (m *Manager) joinGame(elems ...string) string {
	return filepath.Join(append([]string{m.gameRoot()}, elems...)...)
}

// ---- 小文件读写（配置 ini、DLL、Token、Config.json 等）----

// ReadGameFile 读取游戏目录下的文件，返回字节内容。
func (m *Manager) ReadGameFile(elems ...string) ([]byte, error) {
	if m.docker != nil {
		return m.docker.readFile(filepath.Join(elems...))
	}
	return os.ReadFile(m.joinGame(elems...))
}

// WriteGameFile 写入游戏目录下的文件，自动创建父目录。
func (m *Manager) WriteGameFile(data []byte, perm os.FileMode, elems ...string) error {
	if m.docker != nil {
		return m.docker.writeFile(filepath.Join(elems...), data, perm)
	}
	abs := m.joinGame(elems...)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return err
	}
	return os.WriteFile(abs, data, perm)
}

// StatGameFile 返回游戏目录下文件信息。
func (m *Manager) StatGameFile(elems ...string) (os.FileInfo, error) {
	if m.docker != nil {
		return m.docker.stat(filepath.Join(elems...))
	}
	return os.Stat(m.joinGame(elems...))
}

// MkdirAllGame 创建游戏目录（含上级）。
func (m *Manager) MkdirAllGame(perm os.FileMode, elems ...string) error {
	if m.docker != nil {
		return m.docker.mkdirAll(filepath.Join(elems...), perm)
	}
	return os.MkdirAll(m.joinGame(elems...), perm)
}

// RemoveGameFile 删除游戏目录下的文件。
func (m *Manager) RemoveGameFile(elems ...string) error {
	if m.docker != nil {
		return m.docker.remove(filepath.Join(elems...))
	}
	return os.Remove(m.joinGame(elems...))
}

// RemoveAllGame 递归删除游戏目录下的路径。
func (m *Manager) RemoveAllGame(elems ...string) error {
	if m.docker != nil {
		return m.docker.removeAll(filepath.Join(elems...))
	}
	return os.RemoveAll(m.joinGame(elems...))
}

// ListGameDir 列出游戏目录下的条目名称。
func (m *Manager) ListGameDir(elems ...string) ([]string, error) {
	if m.docker != nil {
		out, err := m.docker.execOutput("ls", "-1", "--", m.docker.absPath(filepath.Join(elems...)))
		if err != nil {
			return nil, err
		}
		lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
		if len(lines) == 1 && lines[0] == "" {
			return []string{}, nil
		}
		return lines, nil
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

// DownloadToGame 经 GitHub 镜像下载 URL 到游戏目录下（DLL 等）。
// Docker 下先下载到面板内存/临时文件，再写入容器，避免容器内直连 GitHub 卡住。
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

// ---- 存档目录整树传输（备份、解码、回档）----

const savedRel = "Pal/Saved" // 游戏根到 Saved 目录的相对路径

// FetchSavedToLocal 把游戏的 Pal/Saved 目录复制到本地 localRoot 下，
// 即得到 localRoot/Pal/Saved/...。用于存档备份与 sav_cli 解码。
func (m *Manager) FetchSavedToLocal(localRoot string) error {
	palDir := filepath.Join(localRoot, "Pal")
	if err := os.MkdirAll(palDir, 0755); err != nil {
		return err
	}
	if m.docker != nil {
		// 只拷贝 Pal/Saved（不能 cp 整个 Pal，否则连数 GB 的游戏 exe/DLL 一起打包）。
		// docker cp container:<root>/Pal/Saved - 产生顶层 Saved/ 的 tar，解包到 localRoot/Pal/。
		palDir := filepath.Join(localRoot, "Pal")
		if err := os.MkdirAll(palDir, 0755); err != nil {
			return err
		}
		pr, pw := io.Pipe()
		errCh := make(chan error, 1)
		go func() {
			errCh <- m.docker.tarStreamTo(pw, savedRel)
			_ = pw.Close()
		}()
		err := extractTar(pr, palDir)
		if e := <-errCh; e != nil && err == nil {
			err = e
		}
		return err
	}
	// 本地：递归拷贝
	return copyTree(m.joinGame("Pal", "Saved"), filepath.Join(localRoot, "Pal", "Saved"))
}

// PushLocalToSaved 把本地 localRoot 下的 Pal/Saved 推回游戏目录覆盖。
func (m *Manager) PushLocalToSaved(localRoot string) error {
	localSaved := filepath.Join(localRoot, "Pal", "Saved")
	if m.docker != nil {
		// tar 顶层为 Saved/，docker cp - 解压到容器内 <root>/Pal 下得到 Pal/Saved。
		// 用 docker cp 而非 exec，容器停止状态（回档时已停服）也能写入文件系统。
		pr, pw := io.Pipe()
		errCh := make(chan error, 1)
		go func() {
			errCh <- createTar(pw, localSaved, "Saved")
			_ = pw.Close()
		}()
		err := m.docker.tarStreamFrom(pr, "Pal")
		if e := <-errCh; e != nil && err == nil {
			err = e
		}
		return err
	}
	// 先清空游戏 Saved 再拷回
	target := m.joinGame("Pal", "Saved")
	_ = os.RemoveAll(target)
	return copyTree(localSaved, target)
}

// ---- tar 流辅助 ----

// extractTar 从 r 读取 tar 并解包到 dst 目录。
func extractTar(r io.Reader, dst string) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		// 防 zip-slip
		target := filepath.Join(dst, hdr.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dst)+string(os.PathSeparator)) &&
			filepath.Clean(target) != filepath.Clean(dst) {
			return fmt.Errorf("非法 tar 路径: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(target, os.FileMode(hdr.Mode)&0777)
		case tar.TypeReg:
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
}

// createTar 把 srcDir 目录打包到 w，包内顶层目录名为 topName。
func createTar(w io.Writer, srcDir, topName string) error {
	tw := tar.NewWriter(w)
	defer tw.Close()
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		name := topName
		if rel != "." {
			name = topName + "/" + filepath.ToSlash(rel)
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = name
		if info.IsDir() {
			hdr.Name += "/"
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}

// copyTree 递归拷贝本地目录
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

// GameDisplayPath 返回供前端展示的游戏根路径（容器内路径或本地安装目录）。
func (m *Manager) GameDisplayPath() string { return m.gameRoot() }
