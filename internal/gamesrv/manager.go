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

// Config 游戏服与 SteamCMD 配置（用户在面板填写）
type Config struct {
	// SteamCmdPath SteamCMD 所在目录（也兼容直接填写可执行文件路径）
	// Linux: /root/steamcmd
	// Windows: C:\steamcmd
	SteamCmdPath string `json:"steamcmd_path"`
	// InstallDir 游戏安装目录（SteamCMD 的 force_install_dir）
	InstallDir string `json:"install_dir"`
	// ExtraArgs 服务端额外启动参数（如 -publiclobby -useperfthreads）。
	// 端口等网络参数统一在「游戏配置」中修改，无需在此传入 -port。
	ExtraArgs string `json:"extra_args"`
}

// Status 游戏服状态
type Status struct {
	Installed       bool   `json:"installed"`         // 当前模式所需的服务端已安装
	LinuxInstalled  bool   `json:"linux_installed"`   // 原生 Linux 版已安装
	WindowsInstalled bool  `json:"windows_installed"` // Windows 版已安装
	SteamReady      bool   `json:"steam_ready"`
	Running         bool   `json:"running"`
	Updating        bool   `json:"updating"`
	PID             int    `json:"pid,omitempty"`
	ServerExe       string `json:"server_exe"`
	LinuxExe        string `json:"linux_exe"`
	WindowsExe      string `json:"windows_exe"`
	SteamExe        string `json:"steam_exe"`
	InstallDir      string `json:"install_dir"`
	WineMode        bool   `json:"wine_mode"` // 当前是否 PalDefender/Wine 模式
	State           string `json:"state,omitempty"`
}

// Manager 管理游戏服进程与 SteamCMD
type Manager struct {
	cfg       Config
	serverCmd *exec.Cmd
	updateCmd *exec.Cmd
	logBuf    *ringLog
	// getSetting 读取面板动态设置（由 deps 注入），用于判断 Wine 模式等
	getSetting func(string) string
}

func NewManager() *Manager {
	return &Manager{logBuf: newRingLog(200)}
}

func (m *Manager) SetConfig(cfg Config) {
	m.cfg = cfg
}
func (m *Manager) ConfigValue() Config { return m.cfg }
func (m *Manager) Available() bool    { return true }

// SetSettingGetter 注入面板设置读取函数
func (m *Manager) SetSettingGetter(f func(string) string) {
	m.getSetting = f
}

// steamCmdExe 返回 SteamCMD 可执行文件路径：
// 若配置的是目录，则在目录下查找 steamcmd.sh/steamcmd.exe；
// 若直接配置的是可执行文件，则原样返回（兼容旧配置）。
func (m *Manager) steamCmdExe() string {
	p := m.cfg.SteamCmdPath
	if p == "" {
		return ""
	}
	if info, err := os.Stat(p); err == nil && info.IsDir() {
		name := "steamcmd.sh"
		if runtime.GOOS == "windows" {
			name = "steamcmd.exe"
		}
		return filepath.Join(p, name)
	}
	return p
}

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
	}
	st.SteamExe = m.steamCmdExe()
	// 两个版本的服务端路径都计算
	if runtime.GOOS != "windows" {
		st.LinuxExe = filepath.Join(m.cfg.InstallDir, "PalServer.sh")
	} else {
		st.LinuxExe = ""
	}
	st.WindowsExe = m.winServerExePath()
	st.WineMode = m.wineModeEnabled()
	// 当前模式实际使用的 exe
	if st.WineMode {
		st.ServerExe = st.WindowsExe
	} else {
		st.ServerExe = m.serverExePath()
	}

	// steamcmd 是否存在
	if st.SteamExe != "" {
		if info, err := os.Stat(st.SteamExe); err == nil && !info.IsDir() {
			st.SteamReady = true
		}
	}
	// Linux 版是否已安装
	if st.LinuxExe != "" {
		if info, err := os.Stat(st.LinuxExe); err == nil && !info.IsDir() {
			st.LinuxInstalled = true
		}
	}
	// Windows 版是否已安装
	if st.WindowsExe != "" {
		if info, err := os.Stat(st.WindowsExe); err == nil && !info.IsDir() {
			st.WindowsInstalled = true
		}
	}
	// 当前模式所需版本是否已安装
	st.Installed = st.WineMode && st.WindowsInstalled || !st.WineMode && st.LinuxInstalled
	// 是否在更新
	if m.updateCmd != nil && m.updateCmd.Process != nil && m.isAlive(m.updateCmd.Process.Pid) {
		st.Updating = true
	}
	// 服务端是否在运行（优先检测面板启动的进程，兜底扫描系统进程）
	if m.serverCmd != nil && m.serverCmd.Process != nil && m.isAlive(m.serverCmd.Process.Pid) {
		st.Running = true
		st.PID = m.serverCmd.Process.Pid
	} else if pids := m.findRunningProcesses(); len(pids) > 0 {
		st.Running = true
		st.PID = pids[0]
	}
	if st.Running {
		st.State = "running"
	} else if st.Updating {
		st.State = "updating"
	} else {
		st.State = "stopped"
	}
	return st, nil
}

// Install 用 SteamCMD 安装/更新游戏服（阻塞直到完成，实时输出日志）。
// platform 为 "windows" 时下载 Windows 版服务端，其他（含空）下载当前平台/Linux 版。
func (m *Manager) Install(platform string) error {
	steamExe := m.steamCmdExe()
	if steamExe == "" {
		return errors.New("未配置 SteamCMD 路径")
	}
	if m.cfg.InstallDir == "" {
		return errors.New("未配置游戏安装目录")
	}
	if info, err := os.Stat(steamExe); err != nil || info.IsDir() {
		return fmt.Errorf("SteamCMD 不存在: %s（请确认 SteamCMD 目录下含有 steamcmd.sh/steamcmd.exe）", steamExe)
	}
	if m.isUpdating() {
		return errors.New("正在安装/更新中")
	}
	windows := strings.EqualFold(platform, "windows")
	// 安装 Windows 版需要 Wine（用于后续运行）
	if windows {
		if _, err := exec.LookPath("wine64"); err != nil {
			if _, err2 := exec.LookPath("wine"); err2 != nil {
				return errors.New("安装 Windows 版需要先安装 Wine（面板 PalDefender 页可一键安装）")
			}
		}
	}
	if err := os.MkdirAll(m.cfg.InstallDir, 0755); err != nil {
		return fmt.Errorf("创建安装目录失败: %w", err)
	}

	steamDir := filepath.Dir(steamExe)

	// 首次运行 SteamCMD 需要先完成自更新与配置初始化，否则 app_update 会报 Missing configuration
	m.logBuf.WriteString("=== SteamCMD 初始化（自更新）===\n")
	initCmd := exec.Command(steamExe, "+login", "anonymous", "+quit")
	initCmd.Dir = steamDir
	initCmd.SysProcAttr = newSysProcAttr(true)
	initOut, _ := initCmd.StdoutPipe()
	initCmd.Stderr = initCmd.Stdout
	if err := initCmd.Start(); err != nil {
		return fmt.Errorf("启动 SteamCMD 初始化失败: %w", err)
	}
	go m.pipeLog(initOut)
	if err := initCmd.Wait(); err != nil {
		m.logBuf.WriteString(fmt.Sprintf("警告: SteamCMD 初始化返回错误: %v（继续尝试安装）\n", err))
	}

	// steamcmd +force_install_dir <dir> +login anonymous +app_update 2394010 validate +quit
	// 按用户选择的版本下载（windows 强制 Windows 版，否则当前平台/Linux 版）
	args := []string{
		"+force_install_dir", m.cfg.InstallDir,
		"+login", "anonymous",
	}
	if windows {
		m.logBuf.WriteString("安装 Windows 版服务端（Wine/PalDefender）\n")
		args = append(args, "+@sSteamCmdForcePlatformType", "windows")
	} else if runtime.GOOS != "windows" {
		m.logBuf.WriteString("安装 Linux 原生版服务端\n")
	}
	args = append(args,
		"+app_update", steamAppID, "validate",
		"+quit",
	)
	cmd := exec.Command(steamExe, args...)
	cmd.Dir = steamDir
	cmd.SysProcAttr = newSysProcAttr(true)

	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 SteamCMD 失败: %w", err)
	}
	m.updateCmd = cmd
	m.logBuf.WriteString(fmt.Sprintf("=== SteamCMD 开始安装/更新 (app %s) ===\n", steamAppID))

	// 实时收集输出
	go m.pipeLog(stdout)

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

// Start 启动游戏服。
// 面板以 root 运行，但 PalServer 拒绝 root 启动，因此：
//   - 若面板以 root 运行：查找安装目录属主用户，以该用户身份启动（su -c）；
//   - 若目录属主就是 root：自动创建/使用 paladmin 用户并 chown。
//   - 若面板非 root：直接启动。
func (m *Manager) Start() error {
	// 先确保没有其他游戏服实例在运行
	if running := m.findRunningProcesses(); len(running) > 0 {
		m.logBuf.WriteString(fmt.Sprintf("检测到 %d 个正在运行的游戏服进程，先停止...\n", len(running)))
		if err := m.killAllPalServer(); err != nil {
			m.logBuf.WriteString(fmt.Sprintf("停止旧进程失败: %v\n", err))
		}
		time.Sleep(3 * time.Second)
	}

	args := []string{}
	if m.cfg.ExtraArgs != "" {
		args = append(args, strings.Fields(m.cfg.ExtraArgs)...)
	}

	runUser := ""
	var err error
	if runtime.GOOS != "windows" && os.Geteuid() == 0 {
		runUser, err = m.ensureRunUser()
		if err != nil {
			return fmt.Errorf("准备运行用户失败: %w", err)
		}
	}

	// 是否用 Wine 启动 Windows 版服务端（由面板 PalDefender 模式开关决定）。
	// 开启模式时缺任何前置条件都直接报错，不回退原生启动。
	useWine, wineErr := m.shouldUseWine()
	if wineErr != nil {
		return wineErr
	}
	var exe string
	if useWine {
		exe = m.winServerExePath()
		if _, statErr := os.Stat(exe); statErr != nil {
			return fmt.Errorf("Wine 模式：未找到 Windows 版服务端 %s", exe)
		}
		m.logBuf.WriteString("PalDefender 模式：使用 Wine 启动 Windows 版服务端\n")
	} else {
		exe = m.serverExePath()
		if exe == "" {
			return errors.New("未配置安装目录")
		}
		if _, statErr := os.Stat(exe); statErr != nil {
			return errors.New("服务端未安装，请先在面板点击安装")
		}
	}

	// 构建启动命令
	var cmd *exec.Cmd
	if useWine && runtime.GOOS != "windows" {
		// Wine 启动：wine PalServer-Win64-Shipping-Cmd.exe <args>
		wineExe := "wine64"
		if p, lookErr := exec.LookPath("wine64"); lookErr != nil {
			if p2, lookErr2 := exec.LookPath("wine"); lookErr2 == nil {
				wineExe = p2
			} else {
				return errors.New("未找到 wine/wine64，请先安装 Wine")
			}
		} else {
			wineExe = p
		}
		wineArgs := append([]string{exe}, args...)
		if runUser != "" {
			shellCmd := fmt.Sprintf("cd %s && exec %s %s",
				shellQuote(filepath.Dir(exe)),
				shellQuote(wineExe),
				strings.Join(wineArgs, " "))
			cmd = exec.Command("su", "-s", "/bin/bash", runUser, "-c", shellCmd)
		} else {
			cmd = exec.Command(wineExe, wineArgs...)
		}
		cmd.Dir = filepath.Dir(exe)
		// Wine 需要的环境变量
		cmd.Env = append(os.Environ(),
			"WINEDEBUG=-all",
			"WINEDLLOVERRIDES=d3d9=n,b",
		)
		m.logBuf.WriteString(fmt.Sprintf("Wine 启动: %s %s\n", wineExe, exe))
	} else if runUser != "" {
		shellCmd := fmt.Sprintf("cd %s && exec %s %s",
			shellQuote(filepath.Dir(exe)),
			shellQuote(exe),
			strings.Join(args, " "))
		cmd = exec.Command("su", "-s", "/bin/bash", runUser, "-c", shellCmd)
		cmd.Dir = filepath.Dir(exe)
		m.logBuf.WriteString(fmt.Sprintf("以用户 %s 启动游戏服\n", runUser))
	} else {
		cmd = exec.Command(exe, args...)
		cmd.Dir = filepath.Dir(exe)
	}
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

	time.Sleep(5 * time.Second)
	if !m.isAlive(cmd.Process.Pid) {
		return errors.New("进程启动后立即退出，请检查日志")
	}
	return nil
}

// wineModeEnabled 仅根据面板设置判断是否开启了 PalDefender/Wine 模式（不校验前置条件）。
func (m *Manager) wineModeEnabled() bool {
	return runtime.GOOS != "windows" && m.getSetting != nil &&
		strings.EqualFold(m.getSetting("paldefender.wine_mode"), "true")
}

// shouldUseWine 判断是否用 Wine 启动 Windows 版服务端。
// 未开启 PalDefender 模式时返回 (false, nil)（用原生 Linux 启动）。
// 开启模式时逐项校验前置条件，缺任何一项都返回错误（阻止启动），
// 错误信息说明缺什么以及去哪里安装/下载。
func (m *Manager) shouldUseWine() (bool, error) {
	if runtime.GOOS == "windows" {
		return false, nil
	}
	if !m.wineModeEnabled() {
		return false, nil
	}
	// 1) Wine
	if _, err := exec.LookPath("wine64"); err != nil {
		if _, err2 := exec.LookPath("wine"); err2 != nil {
			return false, errors.New("PalDefender 模式需要 Wine：请在「PalDefender」页点击「一键安装 Wine」")
		}
	}
	// 2) Windows 版服务端 exe
	if m.cfg.InstallDir == "" {
		return false, errors.New("PalDefender 模式需要 Windows 版服务端：请先配置游戏安装目录")
	}
	if _, err := os.Stat(m.winServerExePath()); err != nil {
		return false, errors.New("PalDefender 模式需要 Windows 版服务端：请在「游戏服」页点击「安装/更新」（将自动下载 Windows 版）")
	}
	// 3) PalDefender DLL（d3d9.dll + PalDefender.dll）
	win64 := filepath.Join(m.cfg.InstallDir, "Pal", "Binaries", "Win64")
	missing := []string{}
	if _, err := os.Stat(filepath.Join(win64, "d3d9.dll")); err != nil {
		missing = append(missing, "d3d9.dll")
	}
	if _, err := os.Stat(filepath.Join(win64, "PalDefender.dll")); err != nil {
		missing = append(missing, "PalDefender.dll")
	}
	if len(missing) > 0 {
		return false, fmt.Errorf("PalDefender 模式缺少 %s：请在「PalDefender」页点击「安装 PalDefender」", strings.Join(missing, "、"))
	}
	return true, nil
}

// winServerExePath 返回 Windows 版服务端路径
func (m *Manager) winServerExePath() string {
	if m.cfg.InstallDir == "" {
		return ""
	}
	return filepath.Join(m.cfg.InstallDir, "Pal", "Binaries", "Win64", "PalServer-Win64-Shipping-Cmd.exe")
}

// ensureRunUser 确定启动游戏服使用的非 root 用户：
// 优先使用游戏安装目录的属主；若属主为 root，则创建 paladmin 用户并 chown。
func (m *Manager) ensureRunUser() (string, error) {
	if runtime.GOOS == "windows" {
		return "", nil
	}
	installDir := m.cfg.InstallDir
	if installDir == "" {
		return "", errors.New("未配置游戏安装目录")
	}

	// 查看目录属主
	ownerUID := ""
	if fi, err := os.Stat(installDir); err == nil {
		if stat, ok := fileOwner(fi); ok {
			ownerUID = stat
		}
	}

	// 属主非 root，查找对应用户名
	if ownerUID != "" && ownerUID != "0" {
		if name, err := lookupUsernameByUID(ownerUID); err == nil && name != "" && name != "root" {
			return name, nil
		}
	}

	// 属主是 root 或找不到，使用 paladmin
	const gameUser = "paladmin"
	if _, err := exec.LookPath("id"); err == nil {
		out, err := exec.Command("id", "-u", gameUser).Output()
		if err != nil || strings.TrimSpace(string(out)) == "" {
			// 创建用户
			m.logBuf.WriteString("创建 paladmin 用户用于运行游戏服\n")
			cmd := exec.Command("useradd", "-r", "-m", "-d", installDir, "-s", "/bin/bash", gameUser)
			if out, err := cmd.CombinedOutput(); err != nil {
				return "", fmt.Errorf("创建用户失败: %v: %s", err, strings.TrimSpace(string(out)))
			}
		}
	}
	// chown 游戏目录给 paladmin
	m.logBuf.WriteString(fmt.Sprintf("将 %s 属主改为 %s\n", installDir, gameUser))
	cmd := exec.Command("chown", "-R", gameUser+":"+gameUser, installDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("chown 失败: %v: %s", err, strings.TrimSpace(string(out)))
	}
	// SteamCMD 目录也需要属主可访问（用于安装/更新）
	steamExe := m.steamCmdExe()
	if steamExe != "" {
		steamDir := filepath.Dir(steamExe)
		if steamDir != installDir {
			_ = exec.Command("chown", "-R", gameUser+":"+gameUser, steamDir).Run()
		}
	}
	return gameUser, nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// Stop 停止游戏服（停止所有 PalServer 进程，包括非面板启动的）
func (m *Manager) Stop() error {
	if m.serverCmd != nil && m.serverCmd.Process != nil && m.isAlive(m.serverCmd.Process.Pid) {
		_ = gracefulStop(m.serverCmd, 10*time.Second)
	}
	// 兜底：杀掉所有残留的 PalServer 进程
	if err := m.killAllPalServer(); err != nil {
		return fmt.Errorf("停止失败: %w", err)
	}
	m.serverCmd = nil
	m.logBuf.WriteString("=== 游戏服已停止 ===\n")
	return nil
}

// Restart 重启
func (m *Manager) Restart() error {
	_ = m.Stop()
	time.Sleep(2 * time.Second)
	return m.Start()
}

// findRunningProcesses 查找正在运行的 PalServer 进程 PID
func (m *Manager) findRunningProcesses() []int {
	if runtime.GOOS == "windows" {
		return nil
	}
	out, err := exec.Command("pgrep", "-f", "PalServer-Linux-Shipping").Output()
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

// killAllPalServer 终止所有 PalServer 进程
func (m *Manager) killAllPalServer() error {
	if runtime.GOOS == "windows" {
		if m.serverCmd != nil && m.serverCmd.Process != nil {
			return m.serverCmd.Process.Kill()
		}
		return nil
	}
	// 先尝试 SIGTERM 优雅退出
	_ = exec.Command("pkill", "-TERM", "-f", "PalServer-Linux-Shipping").Run()
	time.Sleep(3 * time.Second)
	// 仍在运行则 SIGKILL 强杀
	_ = exec.Command("pkill", "-KILL", "-f", "PalServer-Linux-Shipping").Run()
	// 同时停止包装脚本
	_ = exec.Command("pkill", "-KILL", "-f", "PalServer.sh").Run()
	return nil
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
