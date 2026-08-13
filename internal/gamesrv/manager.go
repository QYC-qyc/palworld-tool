// Package gamesrv 通过用户自行安装的 SteamCMD 管理幻兽帕鲁服务端：
// 用 steamcmd 路径执行安装/更新/校验，用服务端可执行文件启动/停止。
package gamesrv

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Palworld Steam APP ID
const steamAppID = "2394010"

const (
	DefaultGamePort = "8211"
	DefaultRCONPort = "25575"
	DefaultRESTPort = "8212"
)

// Config 游戏服与 SteamCMD 配置（用户在面板填写）
type Config struct {
	// SteamCmdPath SteamCMD 可执行文件路径
	// Linux: /home/steam/steamcmd/steamcmd.sh
	// Windows: C:\steamcmd\steamcmd.exe
	SteamCmdPath string `json:"steamcmd_path"`
	// InstallDir 游戏安装目录（SteamCMD 的 force_install_dir）
	InstallDir string `json:"install_dir"`
	// ExtraArgs 服务端额外启动参数
	ExtraArgs string `json:"extra_args"`
	GamePort  string `json:"game_port"`
}

// Status 游戏服状态
type Status struct {
	Installed  bool   `json:"installed"`   // 服务端可执行文件存在
	SteamReady bool   `json:"steam_ready"` // steamcmd 可执行文件存在
	Running    bool   `json:"running"`     // 服务端进程在运行
	Updating   bool   `json:"updating"`    // 正在安装/更新
	PID        int    `json:"pid,omitempty"`
	ServerExe  string `json:"server_exe"`  // 服务端可执行文件路径
	InstallDir string `json:"install_dir"`
	GamePort   string `json:"game_port"`
	State      string `json:"state,omitempty"`
}

// Manager 管理游戏服进程与 SteamCMD
type Manager struct {
	cfg       Config
	serverCmd *exec.Cmd
	updateCmd *exec.Cmd
	logBuf    *ringLog
}

func NewManager() *Manager {
	return &Manager{logBuf: newRingLog(200)}
}

func (m *Manager) SetConfig(cfg Config) {
	if cfg.GamePort == "" {
		cfg.GamePort = DefaultGamePort
	}
	m.cfg = cfg
}
func (m *Manager) ConfigValue() Config { return m.cfg }
func (m *Manager) Available() bool    { return true }

// serverExePath 返回服务端可执行文件完整路径
func (m *Manager) serverExePath() string {
	if m.cfg.InstallDir == "" {
		return ""
	}
	exe := "PalServer.sh"
	if runtime.GOOS == "windows" {
		exe = "PalServer.exe"
	}
	return filepath.Join(m.cfg.InstallDir, exe)
}

// GetStatus 查看状态
func (m *Manager) GetStatus() (*Status, error) {
	st := &Status{
		InstallDir: m.cfg.InstallDir,
		GamePort:   m.cfg.GamePort,
	}
	if st.GamePort == "" {
		st.GamePort = DefaultGamePort
	}
	st.ServerExe = m.serverExePath()

	// steamcmd 是否存在
	if m.cfg.SteamCmdPath != "" {
		if info, err := os.Stat(m.cfg.SteamCmdPath); err == nil && !info.IsDir() {
			st.SteamReady = true
		}
	}
	// 服务端是否已安装
	if st.ServerExe != "" {
		if info, err := os.Stat(st.ServerExe); err == nil && !info.IsDir() {
			st.Installed = true
		}
	}
	// 是否在更新
	if m.updateCmd != nil && m.updateCmd.Process != nil && m.isAlive(m.updateCmd.Process.Pid) {
		st.Updating = true
	}
	// 服务端是否在运行
	if m.serverCmd != nil && m.serverCmd.Process != nil && m.isAlive(m.serverCmd.Process.Pid) {
		st.Running = true
		st.PID = m.serverCmd.Process.Pid
		st.State = "running"
	} else if st.Updating {
		st.State = "updating"
	} else {
		st.State = "stopped"
	}
	return st, nil
}

// Install 用 SteamCMD 安装/更新游戏服（阻塞直到完成，实时输出日志）
func (m *Manager) Install() error {
	if m.cfg.SteamCmdPath == "" {
		return errors.New("未配置 SteamCMD 路径")
	}
	if m.cfg.InstallDir == "" {
		return errors.New("未配置游戏安装目录")
	}
	if info, err := os.Stat(m.cfg.SteamCmdPath); err != nil || info.IsDir() {
		return fmt.Errorf("SteamCMD 不存在: %s", m.cfg.SteamCmdPath)
	}
	if m.isUpdating() {
		return errors.New("正在安装/更新中")
	}
	if err := os.MkdirAll(m.cfg.InstallDir, 0755); err != nil {
		return fmt.Errorf("创建安装目录失败: %w", err)
	}

	// steamcmd +force_install_dir <dir> +login anonymous +app_update 2394010 validate +quit
	args := []string{
		"+force_install_dir", m.cfg.InstallDir,
		"+login", "anonymous",
		"+app_update", steamAppID, "validate",
		"+quit",
	}
	cmd := exec.Command(m.cfg.SteamCmdPath, args...)
	cmd.Dir = filepath.Dir(m.cfg.SteamCmdPath)
	cmd.SysProcAttr = newSysProcAttr(true)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 SteamCMD 失败: %w", err)
	}
	m.updateCmd = cmd
	m.logBuf.WriteString(fmt.Sprintf("=== SteamCMD 开始安装/更新 (app %s) ===\n", steamAppID))

	// 实时收集输出
	go m.pipeLog(stdout)
	go m.pipeLog(stderr)

	go func() {
		_ = cmd.Wait()
		m.logBuf.WriteString("=== SteamCMD 结束 ===\n")
		m.updateCmd = nil
	}()
	return nil
}

func (m *Manager) isUpdating() bool {
	return m.updateCmd != nil && m.updateCmd.Process != nil && m.isAlive(m.updateCmd.Process.Pid)
}

// Start 启动游戏服
func (m *Manager) Start() error {
	exe := m.serverExePath()
	if exe == "" {
		return errors.New("未配置安装目录")
	}
	info, err := os.Stat(exe)
	if err != nil || info.IsDir() {
		return errors.New("服务端未安装，请先在面板点击安装")
	}
	if m.serverCmd != nil && m.serverCmd.Process != nil && m.isAlive(m.serverCmd.Process.Pid) {
		return errors.New("游戏服已在运行")
	}

	args := []string{}
	if m.cfg.ExtraArgs != "" {
		args = append(args, strings.Fields(m.cfg.ExtraArgs)...)
	}

	cmd := exec.Command(exe, args...)
	cmd.Dir = filepath.Dir(exe)
	cmd.SysProcAttr = newSysProcAttr(true)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动失败: %w", err)
	}
	m.serverCmd = cmd
	m.logBuf.WriteString("=== 游戏服启动 ===\n")
	go m.pipeLog(stdout)
	go m.pipeLog(stderr)
	go func() { _ = cmd.Wait(); m.logBuf.WriteString("=== 游戏服已停止 ===\n") }()

	time.Sleep(2 * time.Second)
	if !m.isAlive(cmd.Process.Pid) {
		return errors.New("进程启动后立即退出，请检查日志")
	}
	return nil
}

// Stop 停止游戏服
func (m *Manager) Stop() error {
	if m.serverCmd == nil || m.serverCmd.Process == nil || !m.isAlive(m.serverCmd.Process.Pid) {
		return errors.New("游戏服未运行")
	}
	return gracefulStop(m.serverCmd, 30*time.Second)
}

// Restart 重启
func (m *Manager) Restart() error {
	if m.serverCmd != nil && m.serverCmd.Process != nil && m.isAlive(m.serverCmd.Process.Pid) {
		if err := m.Stop(); err != nil {
			return err
		}
		time.Sleep(2 * time.Second)
	}
	return m.Start()
}

// Logs 返回最近日志
func (m *Manager) Logs(lines int) string {
	return m.logBuf.String()
}

// ---- helpers ----

func (m *Manager) pipeLog(rc io.ReadCloser) {
	scanner := bufio.NewScanner(rc)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		m.logBuf.WriteString(scanner.Text() + "\n")
	}
}

func (m *Manager) isAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return processAlive(pid)
}

// ringLog 简易环形日志缓冲
type ringLog struct {
	lines []string
	cap   int
}

func newRingLog(cap int) *ringLog { return &ringLog{cap: cap} }
func (r *ringLog) WriteString(s string) {
	for _, l := range strings.Split(s, "\n") {
		if l == "" {
			continue
		}
		r.lines = append(r.lines, l)
		if len(r.lines) > r.cap {
			r.lines = r.lines[len(r.lines)-r.cap:]
		}
	}
}
func (r *ringLog) String() string { return strings.Join(r.lines, "\n") }
