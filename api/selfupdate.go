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
	"palworld-panel/internal/compose"
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
	if localDigest == "" {
		// 本地镜像没有 digest 信息（比如本地构建的），无法比较，提示可更新
		return true, imageRef, nil
	}

	// 用 docker manifest inspect 获取远端 digest（自动处理 registry 认证/token）
	remoteDigest, err := remoteImageDigest(bin, imageRef)
	if err != nil || remoteDigest == "" {
		// 查不到远端（网络/认证问题）就不提示更新，避免误报
		return false, imageRef, nil
	}

	if strings.Contains(localDigest, remoteDigest) {
		return false, imageRef, nil
	}
	return true, imageRef, nil
}

// remoteImageDigest 用 docker buildx imagetools inspect（或 manifest inspect）
// 获取远端镜像的 digest，自动使用 docker 的 registry 认证。
func remoteImageDigest(bin, imageRef string) (string, error) {
	// buildx imagetools inspect 输出含 Digest: sha256:...，优先尝试
	out, err := exec.Command(bin, "buildx", "imagetools", "inspect", imageRef).Output()
	if err == nil {
		if d := extractDigest(string(out)); d != "" {
			return d, nil
		}
	}
	// 回退：docker manifest inspect
	cmd := exec.Command(bin, "manifest", "inspect", imageRef)
	cmd.Env = append(os.Environ(), "DOCKER_CLI_EXPERIMENTAL=enabled")
	out, err = cmd.Output()
	if err != nil {
		return "", err
	}
	return extractDigest(string(out)), nil
}

// extractDigest 从 manifest inspect 输出中提取 sha256:... digest
func extractDigest(s string) string {
	// 匹配 "Digest": "sha256:..." 或裸 sha256:...
	idx := strings.Index(s, "sha256:")
	if idx < 0 {
		return ""
	}
	rest := s[idx+7:]
	// 取到非 hex 字符为止
	end := 0
	for end < len(rest) && isHex(rest[end]) {
		end++
	}
	if end == 0 {
		return ""
	}
	return "sha256:" + rest[:end]
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
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
	_, _, tag := parseImageRef(image)
	if tag == "" {
		tag = "latest"
	}
	resp := gin.H{"container": true, "has_update": hasUpdate, "image": image, "latest": tag}
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

// updateComposeFile 用内置最新模板更新本地 compose 文件。
// 会先备份旧文件（.bak），再写入新模板。用户自定义应放在 .env 中，
// compose 文件本身由面板维护。
func updateComposeFile(path string) error {
	tmpl := compose.Template()
	if len(tmpl) == 0 {
		return fmt.Errorf("内置 compose 模板为空")
	}
	// 读取现有文件
	existing, _ := os.ReadFile(path)
	// 内容一致则不写
	if string(existing) == string(tmpl) {
		return nil
	}
	// 备份旧文件
	if len(existing) > 0 {
		_ = os.WriteFile(path+".bak", existing, 0644)
	}
	if err := os.WriteFile(path, tmpl, 0644); err != nil {
		return err
	}
	return nil
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

	// 用内置最新模板更新本地 compose 文件（保留用户 .env 的自定义）
	if err := updateComposeFile(composeFile); err != nil {
		appendLog("更新 compose 文件失败（继续使用现有文件）: " + err.Error())
	} else {
		appendLog("compose 文件已更新到最新版")
	}

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
