//go:build windows

package gamesrv

import (
	"fmt"
	"io"
	"os"
)

// Windows 上不使用 Docker 管控游戏服（面板直接运行游戏进程）。
// dockerCtl 为空实现，保证跨平台编译；available() 永远返回 false，
// 因此 NewManager 不会把它赋给 m.docker。
type dockerCtl struct {
	container string
}

func newDockerCtl(name string) *dockerCtl { return &dockerCtl{container: name} }
func (d *dockerCtl) available() bool      { return false }
func (d *dockerCtl) start() error         { return fmt.Errorf("docker 管控在 Windows 上不可用") }
func (d *dockerCtl) stop() error          { return fmt.Errorf("docker 管控在 Windows 上不可用") }
func (d *dockerCtl) restart() error       { return fmt.Errorf("docker 管控在 Windows 上不可用") }
func (d *dockerCtl) isRunning() bool      { return false }
func (d *dockerCtl) isGameRunning() bool  { return false }
func (d *dockerCtl) installOrUpdate() error {
	return fmt.Errorf("docker 管控在 Windows 上不可用")
}
func (d *dockerCtl) installOrUpdateWithLog(logFn func(string)) error {
	return fmt.Errorf("docker 管控在 Windows 上不可用")
}
func (d *dockerCtl) logs(lines int) (string, error) {
	return "", fmt.Errorf("docker 管控在 Windows 上不可用")
}
func (d *dockerCtl) absPath(rel string) string { return rel }

// 以下文件访问方法在 Windows 上不可达（m.docker 恒为 nil），仅为满足编译。
func (d *dockerCtl) readFile(rel string) ([]byte, error)             { return nil, errWindowsDocker }
func (d *dockerCtl) writeFile(rel string, data []byte, perm os.FileMode) error { return errWindowsDocker }
func (d *dockerCtl) stat(rel string) (os.FileInfo, error)            { return nil, errWindowsDocker }
func (d *dockerCtl) mkdirAll(rel string, perm os.FileMode) error     { return errWindowsDocker }
func (d *dockerCtl) remove(rel string) error                        { return errWindowsDocker }
func (d *dockerCtl) removeAll(rel string) error                     { return errWindowsDocker }
func (d *dockerCtl) execOutput(cmd ...string) ([]byte, error)        { return nil, errWindowsDocker }
func (d *dockerCtl) execInput(stdin io.Reader, cmd ...string) error  { return errWindowsDocker }
func (d *dockerCtl) execRun(cmd ...string) error                     { return errWindowsDocker }
func (d *dockerCtl) tarStreamTo(w io.Writer, sub string) error       { return errWindowsDocker }
func (d *dockerCtl) tarStreamFrom(r io.Reader, sub string) error     { return errWindowsDocker }
func (d *dockerCtl) fileExists(rel string) bool                      { return false }

var errWindowsDocker = fmt.Errorf("docker 管控在 Windows 上不可用")
