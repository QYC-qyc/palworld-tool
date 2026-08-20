package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"paladmin/internal/tool"
	"paladmin/service"
)

func restoreBackup(c *gin.Context) {
	id := c.Param("backup_id")
	backups, err := service.ListBackups(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	var backupPath string
	for _, b := range backups {
		if b.BackupId == id {
			backupPath = b.Path
			break
		}
	}
	if backupPath == "" {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "备份不存在"})
		return
	}
	saveDir := tool.EffectiveSavePath()
	// 异步执行回档（停服需要时间）
	go func() {
		if err := service.RestoreBackup(db, saveDir, backupPath); err != nil {
			_ = c.Error(err)
		}
	}()
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "回档已开始：停服→恢复存档→启服，过程可能需要数十秒",
	})
}
