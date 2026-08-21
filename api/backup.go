package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"palworld-panel/internal/tool"
	"palworld-panel/service"
)

func listBackups(c *gin.Context) {
	backups, err := service.ListBackups(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, backups)
}

func deleteBackup(c *gin.Context) {
	id := c.Param("backup_id")
	backups, err := service.ListBackups(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	var path string
	for _, b := range backups {
		if b.BackupId == id {
			path = b.Path
			break
		}
	}
	if path == "" {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "备份不存在"})
		return
	}
	_ = os.Remove(filepath.Join(tool.BackupDir(), path))
	_ = service.DeleteBackup(db, id)
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}
