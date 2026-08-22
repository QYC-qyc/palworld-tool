package gamesrv

import (
	"io"
	"os"
)

// dockerHost 通过 Docker CLI 管控游戏服容器，实现 Host 接口。
// 它组合现有的 dockerCtl（负责容器操作与文件访问）。
type dockerHost struct {
	ctl *dockerCtl
}

// newDockerHost 创建一个基于 Docker 的 Host。
// containerName 为游戏服容器名（如 palworld-gameserver）。
func newDockerHost(containerName string) *dockerHost {
	return &dockerHost{ctl: newDockerCtl(containerName)}
}

func (h *dockerHost) Kind() string { return "docker" }

// ---- 生命周期 ----

func (h *dockerHost) Start() error      { return h.ctl.startGame() }
func (h *dockerHost) Stop() error       { return h.ctl.stopGame() }
func (h *dockerHost) Restart() error    { return h.ctl.restartGame() }
func (h *dockerHost) IsRunning() bool   { return h.ctl.isGameRunning() }
func (h *dockerHost) IsContainerUp() bool { return h.ctl.isRunning() }

// ---- 文件系统 ----

func (h *dockerHost) ReadFile(rel string) ([]byte, error) {
	return h.ctl.readFile(rel)
}
func (h *dockerHost) WriteFile(rel string, data []byte, perm os.FileMode) error {
	return h.ctl.writeFile(rel, data, perm)
}
func (h *dockerHost) Stat(rel string) (os.FileInfo, error) {
	return h.ctl.stat(rel)
}
func (h *dockerHost) MkdirAll(rel string, perm os.FileMode) error {
	return h.ctl.mkdirAll(rel, perm)
}
func (h *dockerHost) Remove(rel string) error      { return h.ctl.remove(rel) }
func (h *dockerHost) RemoveAll(rel string) error   { return h.ctl.removeAll(rel) }
func (h *dockerHost) ListDir(rel string) ([]string, error) {
	return h.ctl.listDir(rel)
}
func (h *dockerHost) GameRoot() string { return dockerGameRoot }

// ---- 存档 ----

func (h *dockerHost) FetchSaved(localRoot string) error {
	return h.ctl.fetchSaved(localRoot)
}
func (h *dockerHost) PushSaved(localRoot string) error {
	return h.ctl.pushSaved(localRoot)
}

// ---- SteamCMD ----

func (h *dockerHost) InstallSteamCMD(steamDir string) error {
	return h.ctl.installSteamCMD(steamDir)
}
func (h *dockerHost) InstallOrUpdateGame(logFn func(string)) error {
	return h.ctl.installOrUpdateWithLog(logFn)
}

// ---- 日志 ----

func (h *dockerHost) Logs(lines int) string {
	out, _ := h.ctl.logs(lines)
	return out
}

// 编译期保证 io.Writer 相关方法不丢失（FetchSaved/PushSaved 用到）
var _ io.Writer = (*io.PipeWriter)(nil)
