//go:build !windows

package gamesrv

import (
	"fmt"
	"os/exec"
	"strings"
)

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
