package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"paladmin/internal/updater"
)

type updaterAPI struct{}

// checkUpdate 检查是否有新版本
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

// doUpdate 执行更新
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
	service := "paladmin"

	// 异步执行更新，先返回响应
	go func() {
		_ = updater.DoUpdate(rel, installDir, service, func(msg string) {
			// 写入日志（简单打印到标准输出，journalctl 可见）
			// 后续可改为写入面板日志缓冲
		})
	}()

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "开始更新到 " + rel.TagName + "，更新完成后服务将自动重启",
	})
}

// resolveInstallDir 确定面板安装目录：优先可执行文件所在目录
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

// CurrentVersion 暴露当前版本
func CurrentVersion() string {
	return updater.CurrentVersion()
}
