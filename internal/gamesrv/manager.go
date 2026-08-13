// Package gamesrv 通过 Docker API 管理幻兽帕鲁服务端容器的部署、启动、停止、更新。
package gamesrv

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const (
	DefaultImage      = "thijsvanloef/palworld-server-docker:latest"
	DefaultContainer  = "palworld"
	DefaultGamePort   = "8211"
	DefaultRCONPort   = "25575"
	DefaultRESTPort   = "8212"
	DefaultDataDir    = "/www/palworld-tool/game"
)

// Status 容器与游戏服状态
type Status struct {
	Installed  bool   `json:"installed"`   // 镜像是否已拉取
	Running    bool   `json:"running"`     // 容器是否在运行
	Container  string `json:"container"`
	Image      string `json:"image"`
	GamePort   string `json:"game_port"`
	DataDir    string `json:"data_dir"`
	Version    string `json:"version,omitempty"`
	State      string `json:"state,omitempty"`
}

// Config 部署游戏服所需的配置
type Config struct {
	Image        string `json:"image"`
	Container    string `json:"container"`
	AdminPassword string `json:"admin_password"`
	ServerName   string `json:"server_name"`
	GamePort     string `json:"game_port"`
	RCONPort     string `json:"rcon_port"`
	RESTPort     string `json:"rest_port"`
	DataDir      string `json:"data_dir"`
}

func (c *Config) applyDefaults() {
	if c.Image == "" { c.Image = DefaultImage }
	if c.Container == "" { c.Container = DefaultContainer }
	if c.GamePort == "" { c.GamePort = DefaultGamePort }
	if c.RCONPort == "" { c.RCONPort = DefaultRCONPort }
	if c.RESTPort == "" { c.RESTPort = DefaultRESTPort }
	if c.DataDir == "" { c.DataDir = DefaultDataDir }
}

// Manager 封装 Docker 客户端
type Manager struct {
	cli *client.Client
}

// NewManager 连接本机 Docker daemon（通过 /var/run/docker.sock）
func NewManager() (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Manager{cli: cli}, nil
}

// Available Docker 是否可用
func (m *Manager) Available() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := m.cli.Ping(ctx)
	return err == nil
}

// GetStatus 查看游戏服状态
func (m *Manager) GetStatus() (*Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	st := &Status{Container: DefaultContainer, Image: DefaultImage, GamePort: DefaultGamePort, DataDir: DefaultDataDir}

	// 容器状态
	c, err := m.cli.ContainerInspect(ctx, DefaultContainer)
	if err == nil {
		st.Installed = true
		st.Running = c.State != nil && c.State.Running
		if c.State != nil {
			st.State = c.State.Status
		}
		if c.Image != "" {
			st.Version = c.Image
		}
	} else if !client.IsErrNotFound(err) {
		return st, nil
	}

	// 镜像是否存在
	imgs, err := m.cli.ImageList(ctx, types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", DefaultImage)),
	})
	if err == nil && len(imgs) > 0 {
		st.Installed = true
	}
	return st, nil
}

// Install 拉取镜像并创建游戏服容器（首次部署）
func (m *Manager) Install(cfg Config) (string, error) {
	cfg.applyDefaults()

	ctx := context.Background()

	// 镜像已存在则跳过拉取
	if !m.imageExists(cfg.Image) {
		if err := m.pullImage(ctx, cfg.Image); err != nil {
			return "", fmt.Errorf("拉取镜像失败: %w", err)
		}
	}

	// 容器已存在则不重复创建
	if m.containerExists(ctx, cfg.Container) {
		return "容器已存在，可直接启动", nil
	}

	// 确保数据目录存在
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return "", fmt.Errorf("创建数据目录失败: %w", err)
	}

	// 端口
	gamePort, _ := nat.NewPort("udp", cfg.GamePort)
	rconPort, _ := nat.NewPort("tcp", cfg.RCONPort)
	restPort, _ := nat.NewPort("tcp", cfg.RESTPort)

	resp, err := m.cli.ContainerCreate(ctx,
		&container.Config{
			Image: cfg.Image,
			Env: buildEnv(cfg),
			ExposedPorts: nat.PortSet{
				gamePort: {}, rconPort: {}, restPort: {},
			},
		},
		&container.HostConfig{
			PortBindings: nat.PortMap{
				gamePort: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: cfg.GamePort + "/udp"}},
				rconPort: []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: cfg.RCONPort}},
				restPort: []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: cfg.RESTPort}},
			},
			Mounts: []mount.Mount{{
				Type:   mount.TypeBind,
				Source: cfg.DataDir,
				Target: "/palworld",
			}},
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
		},
		nil, nil, cfg.Container,
	)
	if err != nil {
		return "", fmt.Errorf("创建容器失败: %w", err)
	}

	if err := m.cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", fmt.Errorf("启动容器失败: %w", err)
	}

	return "游戏服已部署并启动，首次启动会自动下载安装服务端文件，请等待几分钟", nil
}

func buildEnv(cfg Config) []string {
	env := []string{
		"TZ=Asia/Shanghai",
		"ALWAYS_UPDATE_ON_START=true",
		"MULTITHREAD_ENABLED=true",
		"COMMUNITY_SERVER=false",
		"REST_API_ENABLED=true",
		"REST_API_PORT=" + cfg.RESTPort,
		"RCON_ENABLED=true",
		"RCON_PORT=" + cfg.RCONPort,
		"ADMIN_PASSWORD=" + cfg.AdminPassword,
		"SERVER_NAME=" + cfg.ServerName,
		"MAX_PLAYERS=32",
	}
	return env
}

// Start 启动容器
func (m *Manager) Start() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if !m.containerExists(ctx, DefaultContainer) {
		return errors.New("游戏服尚未部署，请先点击安装")
	}
	return m.cli.ContainerStart(ctx, DefaultContainer, types.ContainerStartOptions{})
}

// Stop 停止容器
func (m *Manager) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	timeout := 30
	return m.cli.ContainerStop(ctx, DefaultContainer, container.StopOptions{Timeout: &timeout})
}

// Restart 重启
func (m *Manager) Restart() error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	timeout := 30
	return m.cli.ContainerRestart(ctx, DefaultContainer, container.StopOptions{Timeout: &timeout})
}

// Update 拉取最新镜像并重建容器（保留数据）
func (m *Manager) Update(cfg Config) (string, error) {
	cfg.applyDefaults()
	ctx := context.Background()

	if err := m.pullImage(ctx, cfg.Image); err != nil {
		return "", fmt.Errorf("更新镜像失败: %w", err)
	}
	// 停止并删除旧容器（数据在挂载卷中，不丢失）
	if m.containerExists(ctx, cfg.Container) {
		timeout := 30
		_ = m.cli.ContainerStop(ctx, cfg.Container, container.StopOptions{Timeout: &timeout})
		if err := m.cli.ContainerRemove(ctx, cfg.Container, types.ContainerRemoveOptions{Force: true}); err != nil {
			return "", fmt.Errorf("删除旧容器失败: %w", err)
		}
	}
	return m.Install(cfg)
}

// Logs 获取最近日志
func (m *Manager) Logs(lines int) (string, error) {
	ctx := context.Background()
	if !m.containerExists(ctx, DefaultContainer) {
		return "", errors.New("容器不存在")
	}
	if lines <= 0 {
		lines = 100
	}
	opts := types.ContainerLogsOptions{
		ShowStdout: true, ShowStderr: true, Tail: fmt.Sprintf("%d", lines),
	}
	rc, err := m.cli.ContainerLogs(ctx, DefaultContainer, opts)
	if err != nil {
		return "", err
	}
	defer rc.Close()
	buf := new(strings.Builder)
	_, err = io.Copy(buf, rc)
	return buf.String(), err
}

// ---- helpers ----

func (m *Manager) imageExists(ref string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	imgs, err := m.cli.ImageList(ctx, types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", ref)),
	})
	return err == nil && len(imgs) > 0
}

func (m *Manager) containerExists(ctx context.Context, name string) bool {
	_, err := m.cli.ContainerInspect(ctx, name)
	return err == nil
}

func (m *Manager) pullImage(ctx context.Context, ref string) error {
	rc, err := m.cli.ImagePull(ctx, ref, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer rc.Close()
	_, err = io.Copy(io.Discard, rc)
	return err
}
