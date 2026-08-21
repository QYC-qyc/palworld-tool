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

// writeFile 写入容器内文件，自动创建父目录（二进制安全）
func (d *dockerCtl) writeFile(rel string, data []byte, perm os.FileMode) error {
	abs := d.absPath(rel)
	// 确保父目录存在
	dir := filepath.Dir(abs)
	if err := d.execRun("mkdir", "-p", dir); err != nil {
		return err
	}
	// 用 cat 重定向写入，经 stdin 传原始字节
	if err := d.execInput(bytes.NewReader(data), "sh", "-c", fmt.Sprintf("cat > %s && chmod %o %s", abs, perm, abs)); err != nil {
		return err
	}
	return nil
}

// stat 返回容器内文件信息
func (d *dockerCtl) stat(rel string) (os.FileInfo, error) {
	abs := d.absPath(rel)
	out, err := d.execOutput("stat", "-c", "%s %f %F", abs)
	if err != nil {
		return nil, &os.PathError{Op: "stat", Path: abs, Err: os.ErrNotExist}
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 3 {
		return nil, &os.PathError{Op: "stat", Path: abs, Err: os.ErrNotExist}
	}
	var size int64
	fmt.Sscanf(fields[0], "%d", &size)
	isDir := strings.Contains(fields[2], "directory")
	return &dockerFileInfo{name: filepath.Base(abs), size: size, isDir: isDir}, nil
}

func (d *dockerCtl) mkdirAll(rel string, perm os.FileMode) error {
	return d.execRun("mkdir", "-p", "-m", fmt.Sprintf("%o", perm), d.absPath(rel))
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

// start 启动游戏服容器
func (d *dockerCtl) start() error {
	return d.run("start")
}

// stop 停止游戏服容器
func (d *dockerCtl) stop() error {
	return d.run("stop", "-t", "30")
}

// restart 重启游戏服容器
func (d *dockerCtl) restart() error {
	return d.run("restart")
}

// isRunning 检查容器是否在运行
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
