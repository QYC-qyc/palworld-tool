package gamesrv

import (
	"os"
	"path/filepath"
)

// bareMetalHost 用于二进制（非 Docker）部署：游戏直接在宿主机通过
// Proton/Wine 运行。它实现 Host 接口。
//
// 注意：当前二进制部署的进程管理逻辑仍在 Manager 中（m.host == nil 分支），
// 本类型作为占位与未来迁移的目标。文件/存档操作直接使用本地文件系统。
type bareMetalHost struct {
	gameDir string
}

func newBareMetalHost(gameDir string) *bareMetalHost {
	return &bareMetalHost{gameDir: gameDir}
}

func (h *bareMetalHost) Kind() string { return "baremetal" }

// ---- 生命周期（暂由 Manager 的本地分支处理）----

func (h *bareMetalHost) Start() error      { return errNotImplemented }
func (h *bareMetalHost) Stop() error       { return errNotImplemented }
func (h *bareMetalHost) Restart() error    { return errNotImplemented }
func (h *bareMetalHost) IsRunning() bool   { return false }
func (h *bareMetalHost) IsContainerUp() bool { return true }

// ---- 文件系统：直接本地操作 ----

func (h *bareMetalHost) ReadFile(rel string) ([]byte, error) {
	return os.ReadFile(h.join(rel))
}
func (h *bareMetalHost) WriteFile(rel string, data []byte, perm os.FileMode) error {
	abs := h.join(rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return err
	}
	return os.WriteFile(abs, data, perm)
}
func (h *bareMetalHost) Stat(rel string) (os.FileInfo, error) {
	return os.Stat(h.join(rel))
}
func (h *bareMetalHost) MkdirAll(rel string, perm os.FileMode) error {
	return os.MkdirAll(h.join(rel), perm)
}
func (h *bareMetalHost) Remove(rel string) error    { return os.Remove(h.join(rel)) }
func (h *bareMetalHost) RemoveAll(rel string) error { return os.RemoveAll(h.join(rel)) }
func (h *bareMetalHost) ListDir(rel string) ([]string, error) {
	entries, err := os.ReadDir(h.join(rel))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}
func (h *bareMetalHost) GameRoot() string { return h.gameDir }

// ---- 存档（直接本地拷贝）----

func (h *bareMetalHost) FetchSaved(localRoot string) error {
	return copyTree(filepath.Join(h.gameDir, "Pal", "Saved"), filepath.Join(localRoot, "Pal", "Saved"))
}
func (h *bareMetalHost) PushSaved(localRoot string) error {
	target := filepath.Join(h.gameDir, "Pal", "Saved")
	_ = os.RemoveAll(target)
	return copyTree(filepath.Join(localRoot, "Pal", "Saved"), target)
}

// ---- SteamCMD（由 Manager 本地分支处理）----

func (h *bareMetalHost) InstallSteamCMD(steamDir string) error { return errNotImplemented }
func (h *bareMetalHost) InstallOrUpdateGame(logFn func(string)) error {
	return errNotImplemented
}

// ---- 日志 ----

func (h *bareMetalHost) Logs(lines int) string { return "" }

func (h *bareMetalHost) join(rel string) string {
	return filepath.Join(h.gameDir, filepath.FromSlash(rel))
}

var errNotImplemented = os.ErrInvalid // 占位：该能力暂由 Manager 本地分支处理
