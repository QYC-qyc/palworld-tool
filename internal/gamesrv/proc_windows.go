//go:build windows

package gamesrv

import (
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP | CREATE_NO_WINDOW
const creationFlags = 0x00000008 | 0x00000200 | 0x08000000

func newSysProcAttr(hide bool) *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    hide,
		CreationFlags: creationFlags,
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

// gracefulStop Windows 下先 taskkill 终止进程树，再等待 Wait。
func gracefulStop(cmd *exec.Cmd, timeout time.Duration) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	pid := cmd.Process.Pid
	// taskkill /T 终止进程树
	killCmd := exec.Command("taskkill", "/T", "/PID", strconv.Itoa(pid))
	killCmd.SysProcAttr = newSysProcAttr(true)
	_ = killCmd.Run()

	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		// 强制杀
		forceCmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
		forceCmd.SysProcAttr = newSysProcAttr(true)
		_ = forceCmd.Run()
		<-done
		return nil
	}
}

// diskFreeGB 返回 path 所在分区的可用空间（GB）
func diskFreeGB(path string) (float64, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return 0, err
	}
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var freeBytes, totalBytes, totalFreeBytes uint64
	if err := windows.GetDiskFreeSpaceEx(pathPtr, &freeBytes, &totalBytes, &totalFreeBytes); err != nil {
		return 0, err
	}
	return float64(freeBytes) / 1024 / 1024 / 1024, nil
}

// palProcessName 返回 Windows 游戏进程的映像名。
const palProcessName = "PalServer-Win64-Shipping-Cmd.exe"

// findRunningProcesses 查找正在运行的 PalServer 进程 PID（Windows 用 tasklist）。
func (m *Manager) findRunningProcesses() []int {
	out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq "+palProcessName, "/FO", "CSV", "/NH").Output()
	if err != nil {
		return nil
	}
	var pids []int
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// CSV 格式: "PalServer-Win64-Shipping-Cmd.exe","1234","Console","1","12,345 K"
		parts := strings.Split(line, "\",\"")
		if len(parts) < 2 {
			continue
		}
		pidStr := strings.Trim(parts[1], "\"")
		if pid, err := strconv.Atoi(pidStr); err == nil && pid > 0 {
			pids = append(pids, pid)
		}
	}
	return pids
}

// killAllPalServer 终止所有 PalServer 进程（Windows 用 taskkill /F /IM）。
func (m *Manager) killAllPalServer() error {
	// 先尝试终止面板启动的进程
	if m.serverCmd != nil && m.serverCmd.Process != nil {
		_ = m.serverCmd.Process.Kill()
	}
	cmd := exec.Command("taskkill", "/F", "/IM", palProcessName)
	cmd.SysProcAttr = newSysProcAttr(true)
	_ = cmd.Run()
	return nil
}
