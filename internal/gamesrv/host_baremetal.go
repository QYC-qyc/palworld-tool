//go:build !windows

package gamesrv

import (
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

	ghub "palworld-panel/internal/github"
)

// bareMetalHost 用于二进制（非 Docker）部署：游戏直接在宿主机通过
// Proton 运行。它实现完整的 Host 接口。
type bareMetalHost struct {
	cfg        Config
	getSetting func(string) string
	serverCmd  *exec.Cmd
	updateCmd  *exec.Cmd
	logBuf     *ringLog
}

func newBareMetalHost(cfg Config, getSetting func(string) string, logBuf *ringLog) *bareMetalHost {
	return &bareMetalHost{cfg: cfg, getSetting: getSetting, logBuf: logBuf}
}

func (h *bareMetalHost) Kind() string { return "baremetal" }

// ---- 游戏进程生命周期 ----

func (h *bareMetalHost) IsContainerUp() bool { return true }

func (h *bareMetalHost) IsRunning() bool {
	if h.serverCmd != nil && h.serverCmd.Process != nil && h.isAlive(h.serverCmd.Process.Pid) {
		return true
	}
	if pids := h.findRunningProcesses(); len(pids) > 0 {
		return true
	}
	return false
}

func (h *bareMetalHost) Start() error {
	if h.IsRunning() {
		return errors.New("游戏服已在运行")
	}
	if running := h.findRunningProcesses(); len(running) > 0 {
		h.log("检测到 %d 个正在运行的游戏服进程，先停止...\n", len(running))
		_ = h.killAllPalServer()
		time.Sleep(3 * time.Second)
	}

	if err := h.checkProtonReady(); err != nil {
		return err
	}

	exe := h.winServerExePath()
	if exe == "" {
		return errors.New("未找到游戏可执行文件")
	}

	var args []string
	if h.cfg.ExtraArgs != "" {
		args = append(args, strings.Fields(h.cfg.ExtraArgs)...)
	}

	protonExe := h.protonExePath()
	if protonExe == "" {
		return errors.New("未找到 Proton 可执行文件")
	}
	winDir := h.winInstallDir()
	steamDir := filepath.Dir(h.steamCmdExe())
	protonEnv := append(os.Environ(),
		"PROTON_DIST_PATH="+filepath.Dir(protonExe),
		"PROTON_NO_STEAM=1",
		"STEAM_COMPAT_CLIENT_INSTALL_PATH="+steamDir,
		"STEAM_COMPAT_DATA_PATH="+filepath.Join(winDir, "proton_prefix"),
		"WINEDLLOVERRIDES=d3d9=n,b",
	)

	protonArgs := append([]string{"run", exe}, args...)
	cmd := exec.Command(protonExe, protonArgs...)
	cmd.Dir = filepath.Dir(exe)
	cmd.Env = protonEnv
	cmd.SysProcAttr = newSysProcAttr(true)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动失败: %w", err)
	}
	h.serverCmd = cmd
	h.log("=== 游戏服启动（Proton）===\n")
	go h.pipeLog(stdout)
	go h.pipeLog(stderr)
	go func() { _ = cmd.Wait(); h.log("=== 游戏服已停止 ===\n") }()

	time.Sleep(5 * time.Second)
	if h.serverCmd.Process == nil || !h.isAlive(h.serverCmd.Process.Pid) {
		return errors.New("进程启动后立即退出，请检查日志")
	}
	return nil
}

func (h *bareMetalHost) Stop() error {
	if h.serverCmd != nil && h.serverCmd.Process != nil {
		_ = gracefulStop(h.serverCmd, 10*time.Second)
	}
	if err := h.killAllPalServer(); err != nil {
		return fmt.Errorf("停止失败: %w", err)
	}
	h.serverCmd = nil
	h.log("=== 游戏服已停止 ===\n")
	return nil
}

func (h *bareMetalHost) Restart() error {
	_ = h.Stop()
	time.Sleep(2 * time.Second)
	return h.Start()
}

// ---- 文件系统（直接本地操作）----

func (h *bareMetalHost) ReadFile(rel string) ([]byte, error) {
	return os.ReadFile(h.join(rel))
}
func (h *bareMetalHost) WriteFile(rel string, data []byte, perm os.FileMode) error {
	abs := h.join(rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return err
	}
	return os.WriteFile(abs, data, perm)
}
func (h *bareMetalHost) Stat(rel string) (os.FileInfo, error) { return os.Stat(h.join(rel)) }
func (h *bareMetalHost) MkdirAll(rel string, perm os.FileMode) error {
	return os.MkdirAll(h.join(rel), perm)
}
func (h *bareMetalHost) Remove(rel string) error    { return os.Remove(h.join(rel)) }
func (h *bareMetalHost) RemoveAll(rel string) error { return os.RemoveAll(h.join(rel)) }
func (h *bareMetalHost) ListDir(rel string) ([]string, error) {
	entries, err := os.ReadDir(h.join(rel))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}
func (h *bareMetalHost) GameRoot() string { return h.winInstallDir() }

// ---- 存档 ----

func (h *bareMetalHost) FetchSaved(localRoot string) error {
	return copyTree(filepath.Join(h.winInstallDir(), "Pal", "Saved"),
		filepath.Join(localRoot, "Pal", "Saved"))
}
func (h *bareMetalHost) PushSaved(localRoot string) error {
	target := filepath.Join(h.winInstallDir(), "Pal", "Saved")
	_ = os.RemoveAll(target)
	return copyTree(filepath.Join(localRoot, "Pal", "Saved"), target)
}

// ---- SteamCMD ----

func (h *bareMetalHost) InstallSteamCMD(steamDir string) error {
	return h.installSteamCmdLinux(steamDir)
}
func (h *bareMetalHost) InstallOrUpdateGame(logFn func(string)) error {
	steamExe := h.steamCmdExe()
	if steamExe == "" {
		return errors.New("未配置 SteamCMD 路径")
	}
	bin, err := exec.LookPath(steamExe)
	if err != nil {
		bin = steamExe
	}
	args := []string{"+login", "anonymous",
		"+@sSteamCmdForcePlatformType", "windows",
		"+force_install_dir", h.winInstallDir(),
		"+app_update", steamAppID, "validate", "+quit"}
	cmd := exec.Command(bin, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text() + "\n"
		h.log(line)
		if logFn != nil {
			logFn(line)
		}
	}
	return cmd.Wait()
}

// ---- 日志 ----

func (h *bareMetalHost) Logs(lines int) string { return h.logBuf.Tail(lines) }

// installSteamCmdLinux 下载 steamcmd_linux.tar.gz 并解压到 steamDir。
func (h *bareMetalHost) installSteamCmdLinux(steamDir string) error {
	const url = "https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz"
	h.log("下载 steamcmd_linux.tar.gz...\n")
	tmpFile, err := os.CreateTemp("", "steamcmd-*.tar.gz")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	if err := ghub.DownloadToFile(url, tmpPath); err != nil {
		h.log("镜像下载失败(%v)，尝试 Steam CDN 直连...\n", err)
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

	h.log("解压 SteamCMD...\n")
	cmd := exec.Command("tar", "-xzf", tmpPath, "-C", steamDir)
	cmd.SysProcAttr = newSysProcAttr(true)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("解压 SteamCMD 失败: %w: %s", err, string(out))
	}

	// 首次运行自更新
	h.log("首次运行 SteamCMD 自更新...\n")
	steamExe := filepath.Join(steamDir, "steamcmd.sh")
	initCmd := exec.Command(steamExe, "+login", "anonymous", "+quit")
	initCmd.Dir = steamDir
	initCmd.SysProcAttr = newSysProcAttr(true)
	initOut, _ := initCmd.CombinedOutput()
	h.log(string(initOut) + "\n")

	if _, err := os.Stat(steamExe); err != nil {
		return errors.New("安装完成但未找到 steamcmd.sh，请检查目录权限")
	}
	h.log("SteamCMD 安装完成：%s\n", steamExe)
	return nil
}

// ---- 辅助 ----

func (h *bareMetalHost) join(rel string) string {
	return filepath.Join(h.winInstallDir(), filepath.FromSlash(rel))
}

func (h *bareMetalHost) winInstallDir() string {
	if h.cfg.InstallDir == "" {
		return ""
	}
	return filepath.Join(h.cfg.InstallDir, "PalServer-Win")
}

func (h *bareMetalHost) winServerExePath() string {
	winDir := h.winInstallDir()
	if winDir == "" {
		return ""
	}
	return filepath.Join(winDir, "Pal", "Binaries", "Win64", "PalServer-Win64-Shipping-Cmd.exe")
}

func (h *bareMetalHost) steamCmdExe() string {
	if h.cfg.SteamCmdPath == "" {
		return ""
	}
	if fi, err := os.Stat(h.cfg.SteamCmdPath); err == nil && fi.IsDir() {
		return filepath.Join(h.cfg.SteamCmdPath, "steamcmd.sh")
	}
	return h.cfg.SteamCmdPath
}

func (h *bareMetalHost) protonExePath() string {
	if h.getSetting != nil {
		if p := h.getSetting("proton.path"); p != "" {
			if info, err := os.Stat(p); err == nil && !info.IsDir() {
				return p
			}
		}
	}
	for _, p := range []string{"/opt/GE-Proton/proton", "/usr/bin/proton", "/usr/local/bin/proton"} {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	for _, pat := range []string{
		"/home/*/.steam/steam/compatibilitytools.d/GE-Proton*/proton",
		"/root/.steam/steam/compatibilitytools.d/GE-Proton*/proton",
	} {
		if matches, err := filepath.Glob(pat); err == nil && len(matches) > 0 {
			return matches[len(matches)-1]
		}
	}
	return ""
}

func (h *bareMetalHost) checkProtonReady() error {
	if h.protonExePath() == "" {
		return errors.New("未找到 Proton，请在设置中指定 proton.path")
	}
	if _, err := os.Stat(h.winServerExePath()); err != nil {
		return errors.New("未找到 Windows 版游戏可执行文件，请先安装游戏")
	}
	return nil
}

var palProcessRegex = regexp.MustCompile(`PalServer-Win64|PalServer-Win`)

func (h *bareMetalHost) findRunningProcesses() []int {
	if runtime.GOOS == "windows" {
		return nil
	}
	out, err := exec.Command("pgrep", "-f", "PalServer").Output()
	if err != nil {
		return nil
	}
	var pids []int
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		var pid int
		if _, err := fmt.Sscanf(line, "%d", &pid); err == nil {
			pids = append(pids, pid)
		}
	}
	return pids
}

func (h *bareMetalHost) killAllPalServer() error {
	if runtime.GOOS == "windows" {
		return nil
	}
	_ = exec.Command("pkill", "-f", "PalServer").Run()
	return nil
}

func (h *bareMetalHost) isAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return processAlive(pid)
}

func (h *bareMetalHost) pipeLog(rc io.ReadCloser) {
	scanner := bufio.NewScanner(rc)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		h.log(scanner.Text() + "\n")
	}
}

func (h *bareMetalHost) log(format string, a ...interface{}) {
	if h.logBuf != nil {
		h.logBuf.WriteString(fmt.Sprintf(format, a...))
	}
}
