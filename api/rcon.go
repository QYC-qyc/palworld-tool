package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"paladmin/internal/database"
	"paladmin/internal/tool"
	"paladmin/service"
	"paladmin/service/audit"
)

func listRconCommand(c *gin.Context) {
	cmds, err := service.ListRconCommands(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, cmds)
}

func addRconCommand(c *gin.Context) {
	var cmd database.RconCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	item, err := service.AddRconCommand(db, cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

type sendRconRequest struct {
	Command string `json:"command"`
	Content string `json:"content"`
}

func sendRconCommand(c *gin.Context) {
	var req sendRconRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	full := req.Command
	if req.Content != "" {
		full += " " + req.Content
	}
	resp, err := tool.CustomCommand(full)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	_ = audit.Add(db, "web", "rcon", full, "自定义RCON命令", "success")
	c.JSON(http.StatusOK, gin.H{"response": resp})
}

func putRconCommand(c *gin.Context) {
	id := c.Param("uuid")
	var cmd database.RconCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	if err := service.PutRconCommand(db, id, cmd); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

func removeRconCommand(c *gin.Context) {
	id := c.Param("uuid")
	if err := service.RemoveRconCommand(db, id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}
