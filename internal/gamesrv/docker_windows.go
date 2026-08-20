//go:build windows

package gamesrv

import "fmt"

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
func (d *dockerCtl) installOrUpdate() error {
	return fmt.Errorf("docker 管控在 Windows 上不可用")
}
func (d *dockerCtl) logs(lines int) (string, error) {
	return "", fmt.Errorf("docker 管控在 Windows 上不可用")
}
