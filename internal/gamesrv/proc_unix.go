//go:build !windows

package gamesrv

import (
	"fmt"
	"os/exec"
	"strings"
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

// palProcessPattern 返回 Windows 版游戏进程的 pgrep/pkill 匹配串。
const palProcessPattern = "PalServer-Win64-Shipping-Cmd.exe"

// findRunningProcesses 查找正在运行的 PalServer 进程 PID（Unix 用 pgrep）。
func (m *Manager) findRunningProcesses() []int {
	out, err := exec.Command("pgrep", "-f", palProcessPattern).Output()
	if err != nil {
		return nil
	}
	var pids []int
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		var pid int
		if _, err := fmt.Sscanf(line, "%d", &pid); err == nil && pid > 0 {
			pids = append(pids, pid)
		}
	}
	return pids
}

// killAllPalServer 终止所有 PalServer 进程（Unix 用 pkill）。
func (m *Manager) killAllPalServer() error {
	if m.serverCmd != nil && m.serverCmd.Process != nil {
		_ = m.serverCmd.Process.Kill()
	}
	// 先尝试 SIGTERM 优雅退出
	_ = exec.Command("pkill", "-TERM", "-f", palProcessPattern).Run()
	time.Sleep(3 * time.Second)
	// 仍在运行则 SIGKILL 强杀
	_ = exec.Command("pkill", "-KILL", "-f", palProcessPattern).Run()
	return nil
}
