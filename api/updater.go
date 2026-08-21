package api

import (
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	"paladmin/internal/updater"
)

type updaterAPI struct{}

// updateState 全局更新状态
var updateState struct {
	sync.Mutex
	running  bool
	done     bool
	success  bool
	errMsg   string
	progress updater.Progress
}

func (u *updaterAPI) check(c *gin.Context) {
	rel, hasUpdate, err := updater.Check()
	resp := gin.H{
		"current":   updater.CurrentVersion(),
		"container": inContainer(),
	}
	if err != nil {
		resp["has_update"] = false
		resp["error"] = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp["has_update"] = hasUpdate
	resp["latest"] = rel.TagName
	resp["name"] = rel.Name
	resp["body"] = rel.Body
	c.JSON(http.StatusOK, resp)
}

// do 启动后台更新（容器内禁用：更新应通过拉取新镜像完成）
func (u *updaterAPI) do(c *gin.Context) {
	if inContainer() {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "容器化部署不支持在线更新，请拉取最新镜像后重建容器（docker compose pull && docker compose up -d）"})
		return
	}
	updateState.Lock()
	if updateState.running {
		updateState.Unlock()
		c.JSON(http.StatusOK, gin.H{"running": true})
		return
	}
	updateState.running = true
	updateState.done = false
	updateState.success = false
	updateState.errMsg = ""
	updateState.progress = updater.Progress{Stage: "start", Message: "准备更新...", Percent: 0}
	updateState.Unlock()

	go func() {
		rel, hasUpdate, err := updater.Check()
		if err != nil || !hasUpdate {
			updateState.Lock()
			updateState.running = false
			updateState.done = true
			if err != nil {
				updateState.errMsg = err.Error()
			} else {
				updateState.success = true
				updateState.progress = updater.Progress{Stage: "done", Message: "当前已是最新版本", Percent: 100}
			}
			updateState.Unlock()
			return
		}

		installDir := resolveInstallDir()
		err = updater.DoUpdate(rel, installDir, "paladmin", func(p updater.Progress) {
			updateState.Lock()
			updateState.progress = p
			updateState.Unlock()
		})

		updateState.Lock()
		updateState.running = false
		updateState.done = true
		if err != nil {
			updateState.errMsg = err.Error()
		} else {
			updateState.success = true
			updateState.progress = updater.Progress{Stage: "done", Message: "更新完成，服务重启中...", Percent: 100}
		}
		updateState.Unlock()
	}()

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "开始更新"})
}

// status 返回更新进度
func (u *updaterAPI) status(c *gin.Context) {
	updateState.Lock()
	defer updateState.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"running": updateState.running,
		"done":    updateState.done,
		"success": updateState.success,
		"error":   updateState.errMsg,
		"stage":   updateState.progress.Stage,
		"message": updateState.progress.Message,
		"percent": updateState.progress.Percent,
	})
}

func resolveInstallDir() string {
	exe, err := os.Executable()
	if err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			return filepath.Dir(resolved)
		}
		return filepath.Dir(exe)
	}
	wd, _ := os.Getwd()
	return wd
}
