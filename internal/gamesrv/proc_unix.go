//go:build !windows

package gamesrv

import (
	"os/exec"
	"syscall"
	"time"
)

func newSysProcAttr(hide bool) *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

// processAlive 检查进程是否存活（Unix）
func processAlive(pid int) bool {
	err := syscall.Kill(pid, syscall.Signal(0))
	return err == nil || err == syscall.EPERM
}

// gracefulStop 优雅停止：先 SIGTERM，超时再 SIGKILL
func gracefulStop(cmd *exec.Cmd, timeout time.Duration) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	_ = cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return cmd.Process.Kill()
	}
}
