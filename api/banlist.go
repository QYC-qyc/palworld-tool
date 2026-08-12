package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"paladmin/internal/database"
	"paladmin/service"
	"paladmin/service/anticheat"
)

func listBans(c *gin.Context) {
	activeOnly := c.Query("active") == "1" || c.Query("active") == "true"
	bans, err := service.ListBans(db, activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, bans)
}

type ipRequest struct {
	IP string `json:"ip"`
}

func banIP(c *gin.Context) {
	var req ipRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.IP == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "ip 不能为空"})
		return
	}
	if err := service.AddBan(db, database.BanRecord{
		Type: database.BanIP, Identifier: req.IP,
		Reason: "管理员手动封禁 IP", Issuer: "web",
	}); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	_ = anticheat.AddAudit(db, "web", "banip", req.IP, "手动IP封禁", "success")
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

func unbanIP(c *gin.Context) {
	var req ipRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.IP == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "ip 不能为空"})
		return
	}
	if err := service.RemoveBan(db, database.BanIP, req.IP); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	_ = anticheat.AddAudit(db, "web", "unbanip", req.IP, "手动解封IP", "success")
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}
