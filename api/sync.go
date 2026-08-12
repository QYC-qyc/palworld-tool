package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"paladmin/internal/logger"
	"paladmin/internal/task"
)

func syncData(c *gin.Context) {
	from := c.DefaultQuery("from", "all")
	go func() {
		if from == "rest" || from == "all" {
			task.SyncPlayersOnce()
		}
		if from == "sav" || from == "all" {
			if err := task.SyncSavOnce(); err != nil {
				logger.Errorf("手动存档同步失败: %v", err)
			}
		}
	}()
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}
