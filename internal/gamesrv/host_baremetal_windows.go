//go:build windows

package gamesrv

import (
	"os"
	"path/filepath"
)

// bareMetalHost 在 Windows 上为最小实现（面板本身在 Windows 上
// 通常直接管理本地进程；当前主要目标是 Linux + Proton）。
type bareMetalHost struct {
	gameDir string
}

func newBareMetalHost(cfg Config, getSetting func(string) string, logBuf *ringLog) *bareMetalHost {
	return &bareMetalHost{gameDir: ""}
}

func (h *bareMetalHost) Kind() string                      { return "baremetal" }
func (h *bareMetalHost) Start() error                       { return errNotImpl }
func (h *bareMetalHost) Stop() error                        { return errNotImpl }
func (h *bareMetalHost) Restart() error                     { return errNotImpl }
func (h *bareMetalHost) IsRunning() bool                    { return false }
func (h *bareMetalHost) IsContainerUp() bool                { return true }
func (h *bareMetalHost) ReadFile(rel string) ([]byte, error) {
	return os.ReadFile(h.join(rel))
}
func (h *bareMetalHost) WriteFile(rel string, data []byte, perm os.FileMode) error {
	abs := h.join(rel)
	_ = os.MkdirAll(filepath.Dir(abs), 0755)
	return os.WriteFile(abs, data, perm)
}
func (h *bareMetalHost) Stat(rel string) (os.FileInfo, error) { return os.Stat(h.join(rel)) }
func (h *bareMetalHost) MkdirAll(rel string, perm os.FileMode) error {
	return os.MkdirAll(h.join(rel), perm)
}
func (h *bareMetalHost) Remove(rel string) error          { return os.Remove(h.join(rel)) }
func (h *bareMetalHost) RemoveAll(rel string) error       { return os.RemoveAll(h.join(rel)) }
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
func (h *bareMetalHost) GameRoot() string                          { return h.gameDir }
func (h *bareMetalHost) FetchSaved(localRoot string) error          { return errNotImpl }
func (h *bareMetalHost) PushSaved(localRoot string) error           { return errNotImpl }
func (h *bareMetalHost) InstallSteamCMD(steamDir string) error       { return errNotImpl }
func (h *bareMetalHost) InstallOrUpdateGame(logFn func(string)) error { return errNotImpl }
func (h *bareMetalHost) Logs(lines int) string                      { return "" }
func (h *bareMetalHost) join(rel string) string {
	return filepath.Join(h.gameDir, filepath.FromSlash(rel))
}

var errNotImpl = os.ErrInvalid
