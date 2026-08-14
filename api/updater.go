package api

import (
	"encoding/json"
	"fmt"
	"io"
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

// do 通过 SSE (Server-Sent Events) 推送更新进度
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
	serviceName := "paladmin"

	// 设置 SSE 头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	send := func(p updater.Progress) {
		data, _ := json.Marshal(p)
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		c.Writer.Flush()
	}

	send(updater.Progress{Stage: "start", Message: "开始更新到 " + rel.TagName, Version: rel.TagName})

	if err := updater.DoUpdate(rel, installDir, serviceName, send); err != nil {
		send(updater.Progress{Stage: "error", Message: err.Error()})
		return
	}

	send(updater.Progress{Stage: "done", Message: "更新完成，服务重启中..."})
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

var _ = io.EOF
