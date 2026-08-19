//go:build !windows

package gamesrv

import (
	"os/exec"
	"syscall"
	"time"
)

// diskFreeGB 返回 path 所在分区的可用空间（GB）
func diskFreeGB(path string) (float64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, err
	}
	// Bfree * Bsize = 可用字节
	return float64(st.Bavail) * float64(st.Bsize) / 1024 / 1024 / 1024, nil
}

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
