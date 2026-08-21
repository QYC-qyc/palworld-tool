package api

import (
	"bufio"
	"fmt"
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

// panelImageRef 返回面板容器使用的镜像名:标签（从 compose 项目解析）。
func panelImageRef() (string, error) {
	dir := composeProjectDir()
	cmd := exec.Command("docker", "compose", "-f", filepath.Join(dir, "docker-compose.yml"), "config", "--images")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("解析 compose 镜像失败: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		// 取包含 palworld-panel 的镜像（排除 gameserver）
		if strings.Contains(line, "palworld-panel") {
			return line, nil
		}
	}
	return "", fmt.Errorf("未找到 palworld-panel 镜像配置")
}

// imageHasUpdate 检查 registry 上的镜像 digest 是否与本地不同。
func imageHasUpdate() (bool, string, error) {
	imageRef, err := panelImageRef()
	if err != nil {
		return false, "", err
	}
	bin, err := exec.LookPath("docker")
	if err != nil {
		return false, "", err
	}

	// 本地镜像的 RepoDigests（形如 registry/repo@sha256:...）
	localOut, err := exec.Command(bin, "image", "inspect", imageRef,
		"--format", "{{range .RepoDigests}}{{.}}{{end}}").Output()
	if err != nil {
		// 本地没有该镜像，需要拉取
		return true, imageRef, nil
	}
	localDigest := strings.TrimSpace(string(localOut))

	// 用 registry HTTP API v2 获取远端 manifest digest
	registry, repo, tag := parseImageRef(imageRef)
	if registry == "" || repo == "" {
		return false, imageRef, nil
	}
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry, repo, tag)
	req, _ := http.NewRequest("HEAD", url, nil)
	req.Header.Set("Accept",
		"application/vnd.docker.distribution.manifest.list.v2+json, "+
			"application/vnd.oci.image.index.v1+json, "+
			"application/vnd.docker.distribution.manifest.v2+json, "+
			"application/vnd.oci.image.manifest.v1+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		// 查不到就不提示更新（避免误报）
		return false, imageRef, nil
	}
	remoteDigest := resp.Header.Get("Docker-Content-Digest")
	resp.Body.Close()

	if remoteDigest == "" {
		return false, imageRef, nil
	}
	// 本地 RepoDigests 形如 registry/repo@sha256:xxx
	if strings.Contains(localDigest, remoteDigest) {
		return false, imageRef, nil
	}
	return true, imageRef, nil
}

// parseImageRef 把 "registry/repo:tag" 拆成 registry、repo、tag。
func parseImageRef(ref string) (registry, repo, tag string) {
	tag = "latest"
	if idx := strings.LastIndex(ref, ":"); idx > strings.LastIndex(ref, "/") {
		tag = ref[idx+1:]
		ref = ref[:idx]
	}
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], tag
	}
	return "", ref, tag
}

func selfUpdateCheck(c *gin.Context) {
	if !inContainer() {
		c.JSON(http.StatusOK, gin.H{"has_update": false, "container": false})
		return
	}
	hasUpdate, image, err := imageHasUpdate()
	resp := gin.H{"container": true, "has_update": hasUpdate, "image": image}
	if err != nil {
		resp["error"] = err.Error()
		// 检查失败不阻断，默认无更新
		resp["has_update"] = false
	}
	c.JSON(http.StatusOK, resp)
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

	// 只拉取面板镜像，不动游戏服镜像
	if err := runCmd(dir, "docker", "compose", "-f", composeFile, "pull", "palworld-panel"); err != nil {
		appendLog("拉取面板镜像失败: " + err.Error())
		return
	}
	appendLog("面板镜像拉取完成")

	// 只重建面板容器，--force-recreate 避免旧容器名占用，
	// 不影响 palworld-gameserver 容器。
	if err := runCmd(dir, "docker", "compose", "-f", composeFile, "up", "-d",
		"--force-recreate", "--no-deps", "palworld-panel"); err != nil {
		appendLog("重启面板容器失败: " + err.Error())
		return
	}
	appendLog("更新指令已提交，面板即将重启...")

	// 清理旧的悬空镜像（<none>），释放磁盘空间。
	// 用 setsid 脱离进程组，面板被杀掉后仍继续执行。
	go func() {
		time.Sleep(3 * time.Second)
		prune := exec.Command("docker", "image", "prune", "-f")
		prune.SysProcAttr = sysProcAttrSetsid()
		_ = prune.Run()
	}()

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
