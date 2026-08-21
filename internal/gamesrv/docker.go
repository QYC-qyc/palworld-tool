//go:build !windows

package gamesrv

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// absPath 返回容器内游戏文件的绝对路径
func (d *dockerCtl) absPath(rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(dockerGameRoot, rel)
}

// execOutput 在容器内执行命令并返回 stdout
func (d *dockerCtl) execOutput(cmd ...string) ([]byte, error) {
	bin, err := exec.LookPath("docker")
	if err != nil {
		return nil, fmt.Errorf("未找到 docker: %w", err)
	}
	args := append([]string{"exec", d.container}, cmd...)
	var stdout, stderr bytes.Buffer
	c := exec.Command(bin, args...)
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return nil, fmt.Errorf("docker exec %s 失败: %s: %w", strings.Join(cmd, " "), stderr.String(), err)
	}
	return stdout.Bytes(), nil
}

// fileExists 检查容器内某路径是否存在（文件或目录均可）。
// 容器运行时用 `docker exec test -e`；容器停止时用 `docker cp` 探测（对停止容器也有效）。
func (d *dockerCtl) fileExists(rel string) bool {
	bin, err := exec.LookPath("docker")
	if err != nil {
		return false
	}
	containerPath := d.absPath(rel)

	// 1) 容器运行时：test -e
	if c := exec.Command(bin, "exec", d.container, "test", "-e", containerPath); c.Run() == nil {
		return true
	}
	// 2) 容器可能已停止：用 docker cp 到 /dev/null 探测
	c := exec.Command(bin, "cp", d.container+":"+containerPath, "-")
	c.Stdout = io.Discard
	c.Stderr = io.Discard
	return c.Run() == nil
}

// execInput 在容器内执行命令并通过 stdin 传入数据
func (d *dockerCtl) execInput(stdin io.Reader, cmd ...string) error {
	bin, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("未找到 docker: %w", err)
	}
	args := append([]string{"exec", "-i", d.container}, cmd...)
	var stderr bytes.Buffer
	c := exec.Command(bin, args...)
	c.Stdin = stdin
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("docker exec %s 失败: %s: %w", strings.Join(cmd, " "), stderr.String(), err)
	}
	return nil
}

// readFile 读取容器内文件。优先 docker exec cat（容器运行时）；
// exec 失败（容器停止或文件不存在）时回退到 docker cp，停止的容器也能读取。
func (d *dockerCtl) readFile(rel string) ([]byte, error) {
	abs := d.absPath(rel)
	if out, err := d.execOutput("cat", abs); err == nil {
		return out, nil
	}
	// 回退：docker cp container:path - （输出 tar 流，需解包取出文件内容）
	bin, lookErr := exec.LookPath("docker")
	if lookErr != nil {
		return nil, fmt.Errorf("未找到 docker: %w", lookErr)
	}
	cmd := exec.Command(bin, "cp", d.container+":"+abs, "-")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		// 文件/路径不存在
		if strings.Contains(errMsg, "Could not find the file") ||
			strings.Contains(errMsg, "no such file") ||
			strings.Contains(errMsg, "No such") ||
			strings.Contains(errMsg, "Could not find") {
			return nil, &os.PathError{Op: "cat", Path: abs, Err: os.ErrNotExist}
		}
		return nil, fmt.Errorf("读取容器文件失败: %s: %w", errMsg, err)
	}
	// docker cp 输出 tar，解包取首个 regular 文件的内容
	return extractFirstFileFromTar(stdout.Bytes())
}

// extractFirstFileFromTar 从 tar 流中提取第一个普通文件的内容
func extractFirstFileFromTar(tarData []byte) ([]byte, error) {
	tr := tar.NewReader(bytes.NewReader(tarData))
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("解析 tar 失败: %w", err)
		}
		if hdr.Typeflag == tar.TypeReg {
			return io.ReadAll(tr)
		}
	}
	return nil, &os.PathError{Op: "tar", Path: "", Err: os.ErrNotExist}
}

// writeFile 写入容器内文件，自动创建父目录。
// 优先用 docker cp（容器停止/重启中也能写文件系统），容器运行时再 chmod。
// 回退到 exec（容器运行时）。
func (d *dockerCtl) writeFile(rel string, data []byte, perm os.FileMode) error {
	abs := d.absPath(rel)
	// 构造 tar：包含完整相对目录和文件，docker cp - 会自动创建父目录
	tarData, err := makeTarWithFile(filepath.Base(abs), data)
	if err != nil {
		return err
	}
	// docker cp container:- <tar  ：从 stdin 读取 tar 解压到目标目录
	destDir := filepath.Dir(abs)
	cpArgs := []string{"cp", "-", d.container + ":" + destDir}
	if err := d.runDockerInput(bytes.NewReader(tarData), cpArgs...); err == nil {
		// 容器运行时设置权限；不运行就跳过（cp 已写入文件系统）
		_ = d.execRun("chmod", fmt.Sprintf("%o", perm), abs)
		_ = d.execRun("chown", "steam:steam", abs)
		return nil
	}
	// 回退：容器运行时用 exec 直接写
	if err := d.execRun("mkdir", "-p", destDir); err != nil {
		return err
	}
	return d.execInput(bytes.NewReader(data), "sh", "-c",
		fmt.Sprintf("cat > %s && chmod %o %s", abs, perm, abs))
}

// runDockerInput 执行 docker 命令并通过 stdin 传入数据
func (d *dockerCtl) runDockerInput(stdin io.Reader, args ...string) error {
	bin, err := exec.LookPath("docker")
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, args...)
	cmd.Stdin = stdin
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker %s: %s: %w", strings.Join(args, " "), stderr.String(), err)
	}
	return nil
}

// makeTarWithFile 构造一个 tar，内容为单文件（放在根），供 docker cp 解压
func makeTarWithFile(name string, data []byte) ([]byte, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	hdr := &tar.Header{
		Name: name,
		Mode: 0644,
		Size: int64(len(data)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write(data); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// stat 返回容器内文件信息。容器停止/重启时 exec 会失败，
// 回退到 docker cp 探测文件是否存在（大小未知时返回 0）。
func (d *dockerCtl) stat(rel string) (os.FileInfo, error) {
	abs := d.absPath(rel)
	out, err := d.execOutput("stat", "-c", "%s %f %F", abs)
	if err == nil {
		fields := strings.Fields(strings.TrimSpace(string(out)))
		if len(fields) >= 3 {
			var size int64
			fmt.Sscanf(fields[0], "%d", &size)
			isDir := strings.Contains(fields[2], "directory")
			return &dockerFileInfo{name: filepath.Base(abs), size: size, isDir: isDir}, nil
		}
	}
	// 回退：用 docker cp 探测（容器停止也能检测）
	if d.fileExists(rel) {
		return &dockerFileInfo{name: filepath.Base(abs), size: 0, isDir: false}, nil
	}
	return nil, &os.PathError{Op: "stat", Path: abs, Err: os.ErrNotExist}
}

// mkdirAll 创建目录。容器重启中 exec 会失败；writeFile 的 tar 会自动建目录，
// 因此这里失败不视为致命错误（返回 nil 让上层继续，docker cp 兜底）。
func (d *dockerCtl) mkdirAll(rel string, perm os.FileMode) error {
	_ = d.execRun("mkdir", "-p", "-m", fmt.Sprintf("%o", perm), d.absPath(rel))
	return nil
}

func (d *dockerCtl) remove(rel string) error {
	return d.execRun("rm", "-f", d.absPath(rel))
}

func (d *dockerCtl) removeAll(rel string) error {
	return d.execRun("rm", "-rf", d.absPath(rel))
}

// dockerFileInfo 实现 os.FileInfo（仅满足备份/配置场景所需字段）
type dockerFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (f *dockerFileInfo) Name() string      { return f.name }
func (f *dockerFileInfo) Size() int64       { return f.size }
func (f *dockerFileInfo) Mode() os.FileMode { return 0644 }
func (f *dockerFileInfo) ModTime() time.Time { return time.Time{} }
func (f *dockerFileInfo) IsDir() bool       { return f.isDir }
func (f *dockerFileInfo) Sys() interface{}  { return nil }


// dockerCtl 通过 Docker CLI 管控游戏服容器（容器化部署时使用）。
// 当环境变量 GAMESERVER_CONTAINER 设置为容器名时，Manager 的启停/日志
// 改为操作该容器，而不是直接 exec 游戏进程。
type dockerCtl struct {
	container string
}

func newDockerCtl(name string) *dockerCtl {
	return &dockerCtl{container: name}
}

// available 检查 docker 命令是否可用
func (d *dockerCtl) available() bool {
	_, err := exec.LookPath("docker")
	return err == nil && d.container != ""
}

// start 在游戏服容器内启动游戏进程（容器本身应保持常驻）。
func (d *dockerCtl) start() error {
	if !d.isRunning() {
		// 容器没在跑，先启动容器（它会进入 run 守护模式）
		if err := d.run("start"); err != nil {
			return err
		}
		// 等待容器就绪
		time.Sleep(2 * time.Second)
	}
	// 通知守护进程启动游戏
	_, err := d.execOutput("/home/steam/entrypoint.sh", "start")
	return err
}

// stop 停止容器内的游戏进程（不停止容器）。
func (d *dockerCtl) stop() error {
	if !d.isRunning() {
		return nil
	}
	_, err := d.execOutput("/home/steam/entrypoint.sh", "stop")
	return err
}

// restart 在容器内重启游戏进程（不重启容器）。
func (d *dockerCtl) restart() error {
	if !d.isRunning() {
		return d.start()
	}
	_, err := d.execOutput("/home/steam/entrypoint.sh", "restart")
	return err
}

// isRunning 检查容器是否在运行（容器常驻，应始终为 true）。
func (d *dockerCtl) isRunning() bool {
	bin, err := exec.LookPath("docker")
	if err != nil {
		return false
	}
	out, err := exec.Command(bin, "inspect", "-f", "{{.State.Running}}", d.container).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

// isGameRunning 检查容器内游戏进程（PalServer）是否真的在运行。
// 容器常驻不代表游戏在跑——游戏可能被停止或正在重启。
func (d *dockerCtl) isGameRunning() bool {
	if !d.isRunning() {
		return false
	}
	// 用 entrypoint status 判断（读 PID 文件 + pgrep）
	out, err := d.execOutput("/home/steam/entrypoint.sh", "status")
	if err == nil && strings.Contains(string(out), "running") {
		return true
	}
	// 兜底：直接 pgrep
	// pgrep -f 在进程命令行中匹配 PalServer（兼容 PalServer-Win64-Shipping-Cmd.exe）
	out, err := d.execOutput("pgrep", "-f", "PalServer")
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

// installOrUpdate 在游戏服容器内执行安装/更新（steamcmd app_update）
func (d *dockerCtl) installOrUpdate() error {
	return d.execRun("/home/steam/entrypoint.sh", "update")
}

// logs 返回游戏服容器最近日志
func (d *dockerCtl) logs(lines int) (string, error) {
	bin, err := exec.LookPath("docker")
	if err != nil {
		return "", fmt.Errorf("未找到 docker: %w", err)
	}
	args := []string{"logs", "--tail", fmt.Sprintf("%d", lines), d.container}
	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("docker logs 失败: %w", err)
	}
	return string(out), nil
}

// run 执行 docker 子命令
func (d *dockerCtl) run(args ...string) error {
	bin, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("未找到 docker: %w", err)
	}
	full := append(args, d.container)
	out, err := exec.Command(bin, full...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker %s 失败: %s: %w", strings.Join(args, " "), string(out), err)
	}
	return nil
}

// execRun 在容器内执行命令
func (d *dockerCtl) execRun(cmd ...string) error {
	bin, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("未找到 docker: %w", err)
	}
	args := append([]string{"exec", d.container}, cmd...)
	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker exec %s 失败: %s: %w", strings.Join(cmd, " "), string(out), err)
	}
	return nil
}

// tarStreamTo 把容器内 <root>/<sub> 打包为 tar 流写入 w。
// 使用 `docker cp <container>:<path> -`，它在容器停止时也能读取文件系统（回档停服后仍可备份）。
func (d *dockerCtl) tarStreamTo(w io.Writer, sub string) error {
	bin, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("未找到 docker: %w", err)
	}
	// docker cp container:path -  输出该路径的 tar 流
	args := []string{"cp", d.container + ":" + d.absPath(sub), "-"}
	c := exec.Command(bin, args...)
	c.Stdout = w
	var stderr bytes.Buffer
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("docker cp 读取失败: %s: %w", stderr.String(), err)
	}
	return nil
}

// tarStreamFrom 从 r 读取 tar 流并解压写入容器内 <root>/<sub>。
// 使用 `docker cp - <container>:<path>`，容器停止时也能写入文件系统。
// 注意：docker cp 把 stdin 的 tar 解压到目标路径的**父目录**，因此这里以 sub 作为目标。
func (d *dockerCtl) tarStreamFrom(r io.Reader, sub string) error {
	bin, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("未找到 docker: %w", err)
	}
	args := []string{"cp", "-", d.container + ":" + d.absPath(sub)}
	c := exec.Command(bin, args...)
	c.Stdin = r
	var stderr bytes.Buffer
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("docker cp 写入失败: %s: %w", stderr.String(), err)
	}
	return nil
}
