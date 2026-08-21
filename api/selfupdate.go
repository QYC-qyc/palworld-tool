package api

import (
	"bufio"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// 面板自更新：通过挂载的 docker.sock 执行 `docker compose pull && up -d`。
// 更新过程中面板会被重启，因此命令在后台脱离面板进程运行。

type selfUpdateState struct {
	mu        sync.Mutex
	Running   bool      `json:"running"`
	Done      bool      `json:"done"`
	Success   bool      `json:"success"`
	StartedAt time.Time `json:"started_at,omitempty"`
	Logs      []string  `json:"logs"`
}

var selfUpdate = &selfUpdateState{Logs: []string{}}

const selfUpdateLogMax = 300

func selfUpdateStatus(c *gin.Context) {
	selfUpdate.mu.Lock()
	defer selfUpdate.mu.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"running":    selfUpdate.Running,
		"done":       selfUpdate.Done,
		"success":    selfUpdate.Success,
		"started_at": selfUpdate.StartedAt,
		"logs":       selfUpdate.Logs,
		"container":  inContainer(),
	})
}

func selfUpdateDo(c *gin.Context) {
	if !inContainer() {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "非容器化部署，不支持面板内更新，请手动拉取新二进制/镜像"})
		return
	}
	selfUpdate.mu.Lock()
	if selfUpdate.Running {
		selfUpdate.mu.Unlock()
		c.JSON(http.StatusConflict, ErrorResponse{Error: "更新正在进行中"})
		return
	}
	selfUpdate.Running = true
	selfUpdate.Done = false
	selfUpdate.Success = false
	selfUpdate.StartedAt = time.Now()
	selfUpdate.Logs = []string{}
	selfUpdate.mu.Unlock()

	// 后台执行。用 setsid 让进程在面板被重启后仍继续完成 pull/up。
	go runSelfUpdate()

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "开始更新面板"})
}

func appendLog(line string) {
	selfUpdate.mu.Lock()
	defer selfUpdate.mu.Unlock()
	selfUpdate.Logs = append(selfUpdate.Logs, line)
	if len(selfUpdate.Logs) > selfUpdateLogMax {
		selfUpdate.Logs = selfUpdate.Logs[len(selfUpdate.Logs)-selfUpdateLogMax:]
	}
}

func composeProjectDir() string {
	if d := os.Getenv("COMPOSE_PROJECT_DIR"); d != "" {
		return d
	}
	return "/compose"
}

func runSelfUpdate() {
	defer func() {
		selfUpdate.mu.Lock()
		selfUpdate.Running = false
		selfUpdate.Done = true
		selfUpdate.mu.Unlock()
	}()

	dir := composeProjectDir()
	appendLog("更新目录: " + dir)

	// 找 compose 文件
	composeFile := ""
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			composeFile = filepath.Join(dir, name)
			break
		}
	}
	if composeFile == "" {
		appendLog("错误：未找到 docker-compose.yml，请确认部署目录已挂载到 " + dir)
		return
	}
	appendLog("使用 compose 文件: " + composeFile)

	// docker compose pull
	if err := runCmd(dir, "docker", "compose", "-f", composeFile, "pull"); err != nil {
		appendLog("拉取镜像失败: " + err.Error())
		return
	}
	appendLog("镜像拉取完成")

	// docker compose up -d（会重建面板容器，本进程随后被杀，但 detached 命令已提交）
	if err := runCmd(dir, "docker", "compose", "-f", composeFile, "up", "-d"); err != nil {
		appendLog("重启容器失败: " + err.Error())
		return
	}
	appendLog("更新指令已提交，面板即将重启...")

	selfUpdate.mu.Lock()
	selfUpdate.Success = true
	selfUpdate.mu.Unlock()
}

// runCmd 执行命令并实时收集输出
func runCmd(dir, name string, args ...string) error {
	appendLog("$ " + name + " " + strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	// 关键：让命令脱离当前进程组，面板容器被重启时不被杀掉
	cmd.SysProcAttr = sysProcAttrSetsid()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return err
	}

	reader := bufio.NewReader(stdout)
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			appendLog(strings.TrimRight(line, "\r\n"))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			appendLog("读取输出出错: " + err.Error())
			break
		}
	}
	return cmd.Wait()
}
