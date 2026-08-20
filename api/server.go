package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"paladmin/internal/tool"
)

func getServer(c *gin.Context) {
	info, err := tool.Info()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"version": "", "name": "未连接到服务器", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

func getServerMetrics(c *gin.Context) {
	m, err := tool.Metrics()
	if err != nil {
		// 返回空指标而非 500，避免前端轮询时持续报错；error 字段供调试
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, m)
}

type broadcastRequest struct {
	Message string `json:"message"`
}

func publishBroadcast(c *gin.Context) {
	var req broadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Message == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "message 不能为空"})
		return
	}
	if err := tool.Broadcast(req.Message); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

type shutdownRequest struct {
	Waittime int    `json:"waittime"`
	Message  string `json:"message"`
}

func shutdownServer(c *gin.Context) {
	var req shutdownRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	if err := tool.Shutdown(req.Waittime, req.Message); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}
