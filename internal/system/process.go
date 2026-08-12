package system

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/viper"
)

// ProcessCtl 游戏服务端进程控制接口（回档需要停服/启服）
type ProcessCtl interface {
	Stop() error
	Start() error
	IsRunning() (bool, error)
	Name() string
}

// NewProcessCtl 根据配置返回 systemd / docker / noop 控制器
func NewProcessCtl() ProcessCtl {
	mode := viper.GetString("process.mode")
	switch strings.ToLower(mode) {
	case "systemd":
		return &systemdCtl{service: viper.GetString("process.service")}
	case "docker":
		return &dockerCtl{container: viper.GetString("process.container")}
	default:
		return &noopCtl{}
	}
}

// ---- systemd ----

type systemdCtl struct {
	service string
}

func (s *systemdCtl) Name() string { return "systemd:" + s.service }

func (s *systemdCtl) run(args ...string) error {
	bin, err := exec.LookPath("systemctl")
	if err != nil {
		return fmt.Errorf("未找到 systemctl: %w", err)
	}
	cmd := exec.Command(bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s 失败: %s: %w", strings.Join(args, " "), string(out), err)
	}
	return nil
}

func (s *systemdCtl) Stop() error    { return s.run("stop", s.service) }
func (s *systemdCtl) Start() error   { return s.run("start", s.service) }
func (s *systemdCtl) IsRunning() (bool, error) {
	bin, err := exec.LookPath("systemctl")
	if err != nil {
		return false, err
	}
	out, err := exec.Command(bin, "is-active", s.service).Output()
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(string(out)) == "active", nil
}

// ---- docker ----

type dockerCtl struct {
	container string
}

func (d *dockerCtl) Name() string { return "docker:" + d.container }

func (d *dockerCtl) run(args ...string) error {
	bin, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("未找到 docker: %w", err)
	}
	full := append(args, d.container)
	cmd := exec.Command(bin, full...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker %s 失败: %s: %w", strings.Join(full, " "), string(out), err)
	}
	return nil
}

func (d *dockerCtl) Stop() error  { return d.run("stop") }
func (d *dockerCtl) Start() error { return d.run("start") }
func (d *dockerCtl) IsRunning() (bool, error) {
	bin, err := exec.LookPath("docker")
	if err != nil {
		return false, err
	}
	out, err := exec.Command(bin, "inspect", "-f", "{{.State.Running}}", d.container).Output()
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(string(out)) == "true", nil
}

// ---- noop（未配置进程管理，回档需手动停服） ----

type noopCtl struct{}

func (n *noopCtl) Name() string                  { return "noop(未配置进程管理)" }
func (n *noopCtl) Stop() error                   { return nil }
func (n *noopCtl) Start() error                  { return nil }
func (n *noopCtl) IsRunning() (bool, error)      { return true, nil }
