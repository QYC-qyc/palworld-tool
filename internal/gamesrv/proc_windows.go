//go:build windows

package gamesrv

import (
	"os/exec"
	"syscall"
	"time"
)

func newSysProcAttr(hide bool) *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    hide,
		CreationFlags: 0x00000008 | 0x00000200, // DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP
	}
}

// processAlive 检查进程是否存活（Windows）
func processAlive(pid int) bool {
	h, err := syscall.OpenProcess(0x1000, false, uint32(pid)) // PROCESS_QUERY_LIMITED_INFORMATION
	if err != nil {
		return false
	}
	syscall.CloseHandle(h)
	return true
}

// gracefulStop Windows 下直接终止
func gracefulStop(cmd *exec.Cmd, timeout time.Duration) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}

// diskFreeGB Windows 下不检查（在线更新仅支持 Linux）
func diskFreeGB(path string) (float64, error) {
	return 0, nil
}
