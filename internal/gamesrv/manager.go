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
	"regexp"
	"runtime"
	"strings"
	"time"
)

// Palworld Steam APP ID
const steamAppID = "2394010"

// ansiColorRegex 匹配 ANSI 转义序列（如 SteamCMD 输出的 [0m）
var ansiColorRegex = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

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
	Installed        bool   `json:"installed"`         // Windows 版服务端已安装
	WindowsInstalled bool   `json:"windows_installed"` // Windows 版已安装
	SteamReady       bool   `json:"steam_ready"`
	Running          bool   `json:"running"`
	Updating         bool   `json:"updating"`
	PID              int    `json:"pid,omitempty"`
	ServerExe        string `json:"server_exe"`
	WindowsExe       string `json:"windows_exe"`
	SteamExe         string `json:"steam_exe"`
	InstallDir       string `json:"install_dir"`
	ProtonMode       bool   `json:"proton_mode"` // 当前是否 Proton 模式（始终为 true，保留字段供前端判断）
	State            string `json:"state,omitempty"`
}

// Manager 管理游戏服进程与 SteamCMD
type Manager struct {
	cfg       Config
	serverCmd *exec.Cmd
	updateCmd *exec.Cmd
	logBuf    *ringLog
	// getSetting 读取面板动态设置（由 deps 注入），用于读取 proton.path 等
	getSetting func(string) string
}

func NewManager() *Manager {
	return &Manager{logBuf: newRingLog(200)}
}

func (m *Manager) SetConfig(cfg Config) {
	m.cfg = cfg
}
func (m *Manager) ConfigValue() Config { return m.cfg }
func (m *Manager) Available() bool     { return true }

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

// GetStatus 查看状态
func (m *Manager) GetStatus() (*Status, error) {
	st := &Status{
		InstallDir: m.cfg.InstallDir,
		ProtonMode: runtime.GOOS != "windows",
	}
	st.SteamExe = m.steamCmdExe()
	st.WindowsExe = m.winServerExePath()
	st.ServerExe = st.WindowsExe

	// steamcmd 是否存在
	if st.SteamExe != "" {
		if info, err := os.Stat(st.SteamExe); err == nil && !info.IsDir() {
			st.SteamReady = true
		}
	}
	// Windows 版是否已安装
	if st.WindowsExe != "" {
		if info, err := os.Stat(st.WindowsExe); err == nil && !info.IsDir() {
			st.WindowsInstalled = true
		}
	}
	st.Installed = st.WindowsInstalled
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

// Install 用 SteamCMD 安装/更新 Windows 版游戏服（阻塞直到完成，实时输出日志）。
func (m *Manager) Install() error {
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
	// Windows 版装到独立子目录
	installDir := m.winInstallDir()
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("创建安装目录失败: %w", err)
	}
	// 可写性检查：在目标目录创建临时文件（避免 Disk write failure 才发现）
	if err := checkWritable(installDir); err != nil {
		return fmt.Errorf("安装目录不可写: %w", err)
	}
	// 磁盘空间检查（游戏服务端约需 5GB+）
	if free, err := diskFreeGB(installDir); err == nil {
		m.logBuf.WriteString(fmt.Sprintf("安装目录可用空间：%.1f GB\n", free))
		if free < 5 {
			m.logBuf.WriteString("警告：可用空间不足 5GB，可能导致下载失败\n")
		}
	}

	steamDir := filepath.Dir(steamExe)

	// 首次运行 SteamCMD 需要先完成自更新与配置初始化，否则 app_update 会报 Missing configuration。
	// 关键：初始化时必须带上平台参数 @sSteamCmdForcePlatformType windows，
	// 让 SteamCMD 建立 Windows 平台的配置缓存；否则默认用 Linux 配置，
	// 后续强制 Windows 下载时会冲突报 "Missing configuration"。
	m.logBuf.WriteString("=== SteamCMD 初始化（自更新）===\n")
	initArgs := []string{
		"@sSteamCmdForcePlatformType", "windows",
		"@sSteamCmdForcePlatformBitness", "64",
		"+login", "anonymous",
		"+quit",
	}
	initCmd := exec.Command(steamExe, initArgs...)
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

	// 构造 SteamCMD 参数。始终下载 Windows 64 位版本（通过 Proton 运行）。
	buildArgs := func() []string {
		args := []string{
			"+force_install_dir", installDir,
			"@sSteamCmdForcePlatformType", "windows",
			"@sSteamCmdForcePlatformBitness", "64",
			"+login", "anonymous",
			"+app_update", steamAppID, "validate",
			"+quit",
		}
		return args
	}
	m.logBuf.WriteString(fmt.Sprintf("安装 Windows 版服务端到 %s（Proton/PalDefender）\n", installDir))

	runInstall := func(args []string) error {
		cmd := exec.Command(steamExe, args...)
		cmd.Dir = steamDir
		cmd.SysProcAttr = newSysProcAttr(true)
		stdout, _ := cmd.StdoutPipe()
		cmd.Stderr = cmd.Stdout
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("启动 SteamCMD 失败: %w", err)
		}
		m.updateCmd = cmd
		go m.pipeLog(stdout)
		return cmd.Wait()
	}

	m.logBuf.WriteString(fmt.Sprintf("=== SteamCMD 开始安装/更新 (app %s) ===\n", steamAppID))
	runInit := func() error {
		cmd := exec.Command(steamExe, initArgs...)
		cmd.Dir = steamDir
		cmd.SysProcAttr = newSysProcAttr(true)
		out, _ := cmd.StdoutPipe()
		cmd.Stderr = cmd.Stdout
		if err := cmd.Start(); err != nil {
			return err
		}
		go m.pipeLog(out)
		return cmd.Wait()
	}
	var err error
	err = runInstall(buildArgs())
	if err != nil {
		// Windows 版在 Linux 上下载偶发 Missing configuration / Disk write failure（SteamCMD 平台缓存问题），
		// 清理缓存并重新初始化（带平台参数）后重试一次
		m.logBuf.WriteString("安装失败，清理 SteamCMD 缓存并重新初始化后重试...\n")
		_ = os.RemoveAll(filepath.Join(steamDir, "appcache"))
		_ = os.RemoveAll(filepath.Join(steamDir, "depotcache"))
		_ = os.RemoveAll(filepath.Join(steamDir, "logs"))
		_ = os.RemoveAll(filepath.Join(steamDir, "config"))
		time.Sleep(2 * time.Second)
		_ = runInit()
		err = runInstall(buildArgs())
	}
	m.logBuf.WriteString("=== SteamCMD 结束 ===\n")
	if err != nil {
		m.appendSteamcmdError(steamDir)
		m.updateCmd = nil
		return fmt.Errorf("SteamCMD 安装失败: %w（详见上方日志）", err)
	}
	m.updateCmd = nil
	return nil
}

// appendSteamcmdError 读取 SteamCMD 的 logs/stderr.txt 末尾若干行并写入面板日志，
// 用于诊断 Disk write failure 等错误（其中含具体失败文件路径）。
func (m *Manager) appendSteamcmdError(steamDir string) {
	candidates := []string{
		filepath.Join(steamDir, "logs", "stderr.txt"),
		filepath.Join(steamDir, "..", "logs", "stderr.txt"),
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil || len(data) == 0 {
			continue
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		// 只取最后 15 行
		if len(lines) > 15 {
			lines = lines[len(lines)-15:]
		}
		m.logBuf.WriteString(fmt.Sprintf("--- SteamCMD stderr（%s）---\n", p))
		for _, l := range lines {
			l = strings.TrimSpace(ansiColorRegex.ReplaceAllString(l, ""))
			if l != "" {
				m.logBuf.WriteString(l + "\n")
			}
		}
		return
	}
}

func (m *Manager) isUpdating() bool {
	return m.updateCmd != nil && m.updateCmd.Process != nil && m.isAlive(m.updateCmd.Process.Pid)
}

// protonExePath 返回 Proton 可执行文件路径。
// 优先使用设置 proton.path；为空时按顺序查找常见安装位置。
func (m *Manager) protonExePath() string {
	if m.getSetting != nil {
		if p := m.getSetting("proton.path"); p != "" {
			if info, err := os.Stat(p); err == nil && !info.IsDir() {
				return p
			}
		}
	}
	// 常见固定路径（/opt/GE-Proton 为面板一键安装的 GE-Proton 位置）
	candidates := []string{
		"/opt/GE-Proton/proton",
		"/usr/bin/proton",
		"/usr/local/bin/proton",
	}
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	// GE-Proton 用户目录 glob
	patterns := []string{
		"/home/*/.steam/steam/compatibilitytools.d/GE-Proton*/proton",
		"/root/.steam/steam/compatibilitytools.d/GE-Proton*/proton",
	}
	for _, pat := range patterns {
		matches, err := filepath.Glob(pat)
		if err != nil || len(matches) == 0 {
			continue
		}
		// 返回最后一个（通常字母序最大即最新版本）
		return matches[len(matches)-1]
	}
	return ""
}

// checkProtonReady 校验 Proton 启动所需的全部前置条件：
// a) Proton 可执行文件存在
// b) Windows 版服务端 exe 存在
// c) PalDefender DLL（d3d9.dll/PalDefender.dll）存在
// 缺任何一项返回明确错误。
func (m *Manager) checkProtonReady() error {
	if runtime.GOOS == "windows" {
		return nil
	}
	// a) Proton
	if m.protonExePath() == "" {
		return errors.New("未检测到 Proton，请先在 PalDefender 页一键安装或在设置中指定 Proton 路径")
	}
	// b) Windows 版服务端 exe
	if m.cfg.InstallDir == "" {
		return errors.New("未配置游戏安装目录")
	}
	exe := m.winServerExePath()
	if _, err := os.Stat(exe); err != nil {
		return errors.New("未找到 Windows 版服务端，请先在「游戏服」页点击安装")
	}
	// b2) ARM64 上运行 x86_64 Windows 游戏需要 box64 转译层
	if runtime.GOARCH == "arm64" {
		if _, err := exec.LookPath("box64"); err != nil {
			return errors.New("ARM64 服务器需要 box64 才能运行 x86_64 Windows 游戏服，请在「PalDefender」页一键安装 Proton（会自动安装 box64）或手动安装 box64")
		}
	}
	// c) PalDefender DLL（可选——反作弊未安装时给出警告但不阻止启动，
	//    游戏服本身通过 Proton 即可运行）
	win64 := filepath.Join(m.winInstallDir(), "Pal", "Binaries", "Win64")
	missing := []string{}
	if _, err := os.Stat(filepath.Join(win64, "d3d9.dll")); err != nil {
		missing = append(missing, "d3d9.dll")
	}
	if _, err := os.Stat(filepath.Join(win64, "PalDefender.dll")); err != nil {
		missing = append(missing, "PalDefender.dll")
	}
	if len(missing) > 0 {
		m.logBuf.WriteString(fmt.Sprintf("警告：未安装 PalDefender 反作弊（缺少 %s），游戏服将以 Proton 启动但无反作弊保护。可在「PalDefender」页安装。\n",
			strings.Join(missing, "、")))
	}
	return nil
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

	if err := m.checkProtonReady(); err != nil {
		return err
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

	exe := m.winServerExePath()

	// 构建启动命令
	var cmd *exec.Cmd
	if runtime.GOOS != "windows" {
		// Proton 启动：proton run PalServer-Win64-Shipping-Cmd.exe <args>
		protonExe := m.protonExePath()
		if protonExe == "" {
			return errors.New("未找到 Proton 可执行文件")
		}
		protonArgs := append([]string{"run", exe}, args...)
		winInstallDir := m.winInstallDir()
		steamDir := filepath.Dir(m.steamCmdExe())
		protonEnv := append(os.Environ(),
			"PROTON_DIST_PATH="+filepath.Dir(protonExe),
			"PROTON_NO_STEAM=1",
			"PROTON_NO_ESYNC=1",
			"STEAM_COMPAT_CLIENT_INSTALL_PATH="+steamDir,
			"STEAM_COMPAT_DATA_PATH="+filepath.Join(winInstallDir, "proton_prefix"),
			"WINEDLLOVERRIDES=d3d9=n,b",
		)
		if runUser != "" {
			shellCmd := fmt.Sprintf("cd %s && exec %s",
				shellQuote(filepath.Dir(exe)),
				shellQuote(protonExe)+" "+strings.Join(protonArgs, " "))
			cmd = exec.Command("su", "-s", "/bin/bash", runUser, "-c", shellCmd)
		} else {
			cmd = exec.Command(protonExe, protonArgs...)
		}
		cmd.Dir = filepath.Dir(exe)
		cmd.Env = protonEnv
		m.logBuf.WriteString(fmt.Sprintf("Proton 启动: %s run %s\n", protonExe, exe))
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

// winInstallDir 返回 Windows 版服务端的独立安装目录。
// 默认在游戏安装目录下的 PalServer-Win 子目录。
func (m *Manager) winInstallDir() string {
	if m.cfg.InstallDir == "" {
		return ""
	}
	return filepath.Join(m.cfg.InstallDir, "PalServer-Win")
}

// winServerExePath 返回 Windows 版服务端路径
func (m *Manager) winServerExePath() string {
	winDir := m.winInstallDir()
	if winDir == "" {
		return ""
	}
	return filepath.Join(winDir, "Pal", "Binaries", "Win64", "PalServer-Win64-Shipping-Cmd.exe")
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

// palProcessPattern 返回 Windows 版游戏进程的 pgrep/pkill 匹配串。
// Proton 运行时进程名仍是 PalServer-Win64-Shipping-Cmd.exe。
func (m *Manager) palProcessPattern() string {
	return "PalServer-Win64-Shipping-Cmd.exe"
}

// findRunningProcesses 查找正在运行的 PalServer 进程 PID
func (m *Manager) findRunningProcesses() []int {
	if runtime.GOOS == "windows" {
		return nil
	}
	out, err := exec.Command("pgrep", "-f", m.palProcessPattern()).Output()
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
	pattern := m.palProcessPattern()
	// 先尝试 SIGTERM 优雅退出
	_ = exec.Command("pkill", "-TERM", "-f", pattern).Run()
	time.Sleep(3 * time.Second)
	// 仍在运行则 SIGKILL 强杀
	_ = exec.Command("pkill", "-KILL", "-f", pattern).Run()
	// 兜底：杀掉残留的 Proton 包装进程
	_ = exec.Command("pkill", "-KILL", "-f", "proton.*PalServer").Run()
	return nil
}

// Logs 返回最近日志
func (m *Manager) Logs(lines int) string {
	return m.logBuf.String()
}

// ---- helpers ----

// pipeLog 实时读取子进程输出并写入环形日志。
// SteamCMD 的进度条用 '\r' 回车刷新（无 '\n'），因此按字节读取并把 '\r'、'\n'
// 都作为换行处理，确保下载百分比能实时显示。去掉 ANSI 转义序列。
func (m *Manager) pipeLog(rc io.ReadCloser) {
	reader := bufio.NewReaderSize(rc, 4096)
	var line []byte
	flush := func() {
		s := strings.TrimRight(string(line), " \t\r\n")
		if s != "" {
			// 去除简单 ANSI 转义（如 [0m）
			s = ansiColorRegex.ReplaceAllString(s, "")
			m.logBuf.WriteString(s + "\n")
		}
		line = line[:0]
	}
	for {
		b, err := reader.ReadByte()
		if err != nil {
			flush()
			return
		}
		if b == '\n' || b == '\r' {
			flush()
		} else {
			line = append(line, b)
		}
	}
}

func (m *Manager) isAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return processAlive(pid)
}

// checkWritable 在目录中创建并删除临时文件，验证可写性
func checkWritable(dir string) error {
	f, err := os.CreateTemp(dir, ".paladmin-wtest-*")
	if err != nil {
		return err
	}
	name := f.Name()
	f.Close()
	return os.Remove(name)
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
