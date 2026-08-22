// Package gamesrv 管理游戏服的生命周期、文件和 SteamCMD。
//
// 它通过 Host 接口抽象底层部署方式（Docker 容器或裸机进程），
// Manager 只依赖该接口，不关心游戏跑在容器里还是本地。
package gamesrv

import (
	"os"
)

// Host 是游戏服运行环境的抽象。
// 实现者负责：游戏进程的启停、游戏目录文件读写、SteamCMD 安装与更新、日志。
type Host interface {
	// Kind 返回主机类型标识，如 "docker" 或 "baremetal"。
	Kind() string

	// ---- 游戏进程生命周期 ----

	// Start 启动游戏服。若已在运行应返回错误。
	Start() error
	// Stop 优雅停止游戏服。
	Stop() error
	// Restart 重启游戏服。
	Restart() error
	// IsRunning 报告游戏进程是否真正在运行（不是容器是否在跑）。
	IsRunning() bool
	// IsContainerUp 报告容器/宿主环境是否就绪（Docker 下指容器 Up）。
	// 裸机部署下恒为 true。
	IsContainerUp() bool

	// ---- 游戏文件系统 ----

	// ReadFile 读取游戏目录下的文件（相对路径，用 / 分隔）。
	ReadFile(rel string) ([]byte, error)
	// WriteFile 写入游戏目录下的文件，自动创建父目录。
	WriteFile(rel string, data []byte, perm os.FileMode) error
	// Stat 返回游戏目录下文件信息。
	Stat(rel string) (os.FileInfo, error)
	// MkdirAll 创建游戏目录。
	MkdirAll(rel string, perm os.FileMode) error
	// Remove 删除文件。
	Remove(rel string) error
	// RemoveAll 递归删除路径。
	RemoveAll(rel string) error
	// ListDir 列出目录下条目名称。
	ListDir(rel string) ([]string, error)
	// GameRoot 返回游戏根目录的展示路径（容器内或本地绝对路径）。
	GameRoot() string

	// ---- 存档整树传输（备份/回档用）----

	// FetchSaved 把游戏的 Pal/Saved 目录复制到 localRoot/Pal/Saved。
	FetchSaved(localRoot string) error
	// PushSaved 把 localRoot/Pal/Saved 推回游戏目录覆盖。
	PushSaved(localRoot string) error

	// ---- SteamCMD ----

	// InstallSteamCMD 安装 SteamCMD 到指定目录（幂等）。
	InstallSteamCMD(steamDir string) error
	// InstallOrUpdateGame 下载/更新游戏服，实时日志写入 logFn（可为 nil）。
	InstallOrUpdateGame(logFn func(string)) error

	// ---- 日志 ----

	// Logs 返回最近 lines 行游戏服日志。
	Logs(lines int) string
}

// 编译期检查 Host 实现。
var (
	_ Host = (*dockerHost)(nil)
	_ Host = (*bareMetalHost)(nil)
)
