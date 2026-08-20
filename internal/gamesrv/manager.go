// Package gamesrv 通过用户自行安装的 SteamCMD 管理幻兽帕鲁服务端：
// 用 steamcmd 路径执行安装/更新/校验，用服务端可执行文件启动/停止。
package gamesrv

import (
	"archive/zip"
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	ghub "paladmin/internal/github"
)

// Palworld Steam APP ID
const steamAppID = "2394010"

// ansiColorRegex 匹配 ANSI 转义序列（如 SteamCMD 输出的 [0m）
var ansiColorRegex = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

// Config 游戏服与 SteamCMD 配置（用户在面板填写）
type Config struct {
	// SteamCmdPath SteamCMD 所在目录（也兼容直接填写可执行文件路径）
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
	Installed  bool   `json:"installed"` // 服务端已安装
	SteamReady bool   `json:"steam_ready"`
	Running    bool   `json:"running"`
	Updating   bool   `json:"updating"`
	PID        int    `json:"pid,omitempty"`
	ServerExe  string `json:"server_exe"`
	SteamExe   string `json:"steam_exe"`
	InstallDir string `json:"install_dir"`
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
	m.cfg = cfg
}
func (m *Manager) ConfigValue() Config { return m.cfg }
func (m *Manager) Available() bool     { return true }

// steamCmdExe 返回 SteamCMD 可执行文件路径：
// 若配置的是目录，则在目录下查找 steamcmd.exe（Windows）/steamcmd.sh（Unix）；
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

// serverExePath 返回 Windows 版服务端可执行文件路径
func (m *Manager) serverExePath() string {
	if m.cfg.InstallDir == "" {
		return ""
	}
	return filepath.Join(m.cfg.InstallDir, "Pal", "Binaries", "Win64", "PalServer-Win64-Shipping-Cmd.exe")
}

// GetStatus 查看状态
func (m *Manager) GetStatus() (*Status, error) {
	st := &Status{
		InstallDir: m.cfg.InstallDir,
	}
	st.SteamExe = m.steamCmdExe()
	st.ServerExe = m.serverExePath()

	// steamcmd 是否存在
	if st.SteamExe != "" {
		if info, err := os.Stat(st.SteamExe); err == nil && !info.IsDir() {
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

// InstallSteamCMD 下载并安装 SteamCMD 到配置的 SteamCmdPath 目录（Windows 原生）。
// 如果目录下已存在 steamcmd.exe 则直接返回。
func (m *Manager) InstallSteamCMD() error {
	if m.cfg.SteamCmdPath == "" {
		return errors.New("请先在上方填写 SteamCMD 安装目录")
	}
	// 已安装则跳过
	if exe := m.steamCmdExe(); exe != "" {
		if _, err := os.Stat(exe); err == nil {
			m.logBuf.WriteString(fmt.Sprintf("SteamCMD 已存在：%s\n", exe))
			return nil
		}
	}
	if err := os.MkdirAll(m.cfg.SteamCmdPath, 0755); err != nil {
		return fmt.Errorf("创建 SteamCMD 目录失败: %w", err)
	}

	m.logBuf.WriteString("=== 安装 SteamCMD ===\n")
	return m.installSteamCmdWindows()
}

// installSteamCmdWindows 下载 steamcmd.zip 并解压（Windows 原生面板）
func (m *Manager) installSteamCmdWindows() error {
	const url = "https://steamcdn-a.akamaihd.net/client/installer/steamcmd.zip"
	m.logBuf.WriteString("下载 steamcmd.zip...\n")
	tmpFile, err := os.CreateTemp("", "steamcmd-*.zip")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// 经镜像尝试下载（Steam CDN 直连通常可用，但用 ghub 兜底）
	if err := ghub.DownloadToFile(url, tmpPath); err != nil {
		m.logBuf.WriteString(fmt.Sprintf("镜像下载失败(%v)，尝试 Steam CDN 直连...\n", err))
		resp, err2 := http.Get(url)
		if err2 != nil {
			return fmt.Errorf("下载 SteamCMD 失败: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("下载 SteamCMD 失败: HTTP %d", resp.StatusCode)
		}
		if _, err := io.Copy(tmpFile, resp.Body); err != nil {
			return err
		}
	}
	tmpFile.Close()

	m.logBuf.WriteString("解压 SteamCMD...\n")
	if err := m.unzipSteamCmd(tmpPath); err != nil {
		return fmt.Errorf("解压 SteamCMD 失败: %w", err)
	}

	// 解压后查找 steamcmd.exe（zip 根目录或子目录都兼容）
	steamExe := m.findSteamCmdExe()
	if steamExe == "" {
		// 列出目录内容用于诊断
		entries, _ := os.ReadDir(m.cfg.SteamCmdPath)
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		return fmt.Errorf("安装完成但未找到 steamcmd.exe（目录内容: %s）", strings.Join(names, ", "))
	}
	m.logBuf.WriteString(fmt.Sprintf("找到 steamcmd.exe: %s\n", steamExe))

	// SteamCMD 首次运行需要自更新，跑一次 +login anonymous +quit 完成初始化
	m.logBuf.WriteString("首次运行 SteamCMD 自更新...\n")
	initCmd := exec.Command(steamExe, "+login", "anonymous", "+quit")
	initCmd.Dir = m.cfg.SteamCmdPath
	initCmd.SysProcAttr = newSysProcAttr(true)
	initOut, _ := initCmd.CombinedOutput()
	m.logBuf.WriteString(string(initOut) + "\n")

	m.logBuf.WriteString(fmt.Sprintf("SteamCMD 安装完成：%s\n", steamExe))
	return nil
}

// unzipSteamCmd 用 Go 标准库解压 steamcmd.zip 到 SteamCmdPath（不依赖 PowerShell）。
func (m *Manager) unzipSteamCmd(zipPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	if err := os.MkdirAll(m.cfg.SteamCmdPath, 0o755); err != nil {
		return err
	}
	for _, f := range r.File {
		// 解压到目标目录（steamcmd.zip 内文件在根目录）
		target := filepath.Join(m.cfg.SteamCmdPath, f.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(m.cfg.SteamCmdPath)+string(os.PathSeparator)) {
			return fmt.Errorf("非法路径: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// findSteamCmdExe 在 SteamCmdPath 下查找 steamcmd.exe（根目录或一级子目录）。
func (m *Manager) findSteamCmdExe() string {
	// 先查根目录
	exe := filepath.Join(m.cfg.SteamCmdPath, "steamcmd.exe")
	if info, err := os.Stat(exe); err == nil && !info.IsDir() {
		return exe
	}
	// 递归查找（最多 2 层）
	_ = filepath.WalkDir(m.cfg.SteamCmdPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.EqualFold(d.Name(), "steamcmd.exe") {
			exe = path
			return filepath.SkipAll
		}
		return nil
	})
	if info, err := os.Stat(exe); err == nil && !info.IsDir() {
		return exe
	}
	return ""
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
		return fmt.Errorf("SteamCMD 不存在: %s（请确认 SteamCMD 目录下含有 steamcmd.exe）", steamExe)
	}
	if m.isUpdating() {
		return errors.New("正在安装/更新中")
	}
	installDir := m.cfg.InstallDir
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
	m.logBuf.WriteString("=== SteamCMD 初始化（自更新）===\n")
	initArgs := []string{
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

	// 构造 SteamCMD 参数。Windows 上 steamcmd 默认下载 Windows 版，无需平台参数。
	buildArgs := func() []string {
		return []string{
			"+force_install_dir", installDir,
			"+login", "anonymous",
			"+app_update", steamAppID, "validate",
			"+quit",
		}
	}
	m.logBuf.WriteString(fmt.Sprintf("安装 Windows 版服务端到 %s\n", installDir))

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
		// 偶发 Missing configuration / Disk write failure（SteamCMD 缓存问题），
		// 清理缓存并重新初始化后重试一次
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

// appendSteamcmdError 读取 SteamCMD 的 logs/*.txt 末尾若干行并写入面板日志，
// 用于诊断 Disk write failure 等错误（其中含具体失败文件路径）。
// Windows 下 SteamCMD 日志位于 <SteamCmdPath>/logs/content_log.txt。
func (m *Manager) appendSteamcmdError(steamDir string) {
	logsDir := filepath.Join(steamDir, "logs")
	// 优先 content_log.txt，其次目录下任意 .txt
	candidates := []string{filepath.Join(logsDir, "content_log.txt")}
	if entries, err := os.ReadDir(logsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.HasSuffix(strings.ToLower(name), ".txt") {
				p := filepath.Join(logsDir, name)
				if p != candidates[0] {
					candidates = append(candidates, p)
				}
			}
		}
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
		m.logBuf.WriteString(fmt.Sprintf("--- SteamCMD 日志（%s）---\n", p))
		for _, l := range lines {
			l = strings.TrimSpace(ansiColorRegex.ReplaceAllString(l, ""))
			if l != "" {
				m.logBuf.WriteString(l + "\n")
			}
		}
		return
	}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func (m *Manager) isUpdating() bool {
	return m.updateCmd != nil && m.updateCmd.Process != nil && m.isAlive(m.updateCmd.Process.Pid)
}

// checkReady 校验启动游戏服所需的前置条件：
// a) 已配置安装目录
// b) 服务端 exe 存在
// PalDefender DLL（d3d9.dll/PalDefender.dll）缺失时仅警告，不阻止启动。
func (m *Manager) checkReady() error {
	if m.cfg.InstallDir == "" {
		return errors.New("未配置游戏安装目录")
	}
	exe := m.serverExePath()
	if _, err := os.Stat(exe); err != nil {
		return errors.New("未找到服务端可执行文件，请先在「游戏服」页点击安装")
	}
	// PalDefender DLL 可选——缺失时仅警告
	win64 := filepath.Dir(exe)
	missing := []string{}
	if _, err := os.Stat(filepath.Join(win64, "d3d9.dll")); err != nil {
		missing = append(missing, "d3d9.dll")
	}
	if _, err := os.Stat(filepath.Join(win64, "PalDefender.dll")); err != nil {
		missing = append(missing, "PalDefender.dll")
	}
	if len(missing) > 0 {
		m.logBuf.WriteString(fmt.Sprintf("警告：未安装 PalDefender 反作弊（缺少 %s），游戏服将正常启动但无反作弊保护。可在「PalDefender」页安装。\n",
			strings.Join(missing, "、")))
	}
	return nil
}

// Start 启动游戏服（Windows 原生，直接运行 PalServer-Win64-Shipping-Cmd.exe）。
func (m *Manager) Start() error {
	// 先确保没有其他游戏服实例在运行
	if running := m.findRunningProcesses(); len(running) > 0 {
		m.logBuf.WriteString(fmt.Sprintf("检测到 %d 个正在运行的游戏服进程，先停止...\n", len(running)))
		if err := m.killAllPalServer(); err != nil {
			m.logBuf.WriteString(fmt.Sprintf("停止旧进程失败: %v\n", err))
		}
		time.Sleep(3 * time.Second)
	}

	if err := m.checkReady(); err != nil {
		return err
	}

	args := []string{}
	if m.cfg.ExtraArgs != "" {
		args = append(args, strings.Fields(m.cfg.ExtraArgs)...)
	}

	exe := m.serverExePath()
	cmd := exec.Command(exe, args...)
	cmd.Dir = filepath.Dir(exe)
	cmd.SysProcAttr = newSysProcAttr(true)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动失败: %w", err)
	}
	m.serverCmd = cmd
	m.logBuf.WriteString(fmt.Sprintf("=== 游戏服启动: %s %s ===\n", exe, strings.Join(args, " ")))
	go m.pipeLog(stdout)
	go m.pipeLog(stderr)
	go func() { _ = cmd.Wait(); m.logBuf.WriteString("=== 游戏服已停止 ===\n") }()

	time.Sleep(5 * time.Second)
	if !m.isAlive(cmd.Process.Pid) {
		return errors.New("进程启动后立即退出，请检查日志")
	}
	return nil
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

// Logs 返回最近日志
func (m *Manager) Logs(lines int) string {
	return m.logBuf.String()
}

// WriteLog 写入一行日志到环形日志（供 API 层记录后台任务结果）。
func (m *Manager) WriteLog(s string) {
	m.logBuf.WriteString(s + "\n")
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
