package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"paladmin/internal/updater"
)

type updaterAPI struct{}

func (u *updaterAPI) check(c *gin.Context) {
	rel, hasUpdate, err := updater.Check()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"current": updater.CurrentVersion(), "has_update": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"current":    updater.CurrentVersion(),
		"has_update": hasUpdate,
		"latest":     rel.TagName,
		"name":       rel.Name,
		"body":       rel.Body,
		"published":  rel.PublishedAt,
	})
}

// do 触发更新：执行安装脚本，立即返回，前端轮询 health 等待重启
func (u *updaterAPI) do(c *gin.Context) {
	rel, hasUpdate, err := updater.Check()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	if !hasUpdate {
		c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "当前已是最新版本"})
		return
	}

	installDir := resolveInstallDir()
	// 异步执行更新，不阻塞 HTTP 响应
	go func() {
		_ = updater.DoUpdate(rel, installDir, "paladmin", nil)
	}()

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "开始更新到 " + rel.TagName + "，服务即将重启",
	})
}

// resolveInstallDir 确定面板安装目录
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
