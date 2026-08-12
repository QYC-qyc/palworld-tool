package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"paladmin/internal/database"
	"paladmin/service"
)

func listWhite(c *gin.Context) {
	wl, err := service.ListWhitelist(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, wl)
}

func addWhite(c *gin.Context) {
	var p database.PlayerW
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	if err := service.AddWhitelist(db, p); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

func removeWhite(c *gin.Context) {
	var p database.PlayerW
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	if err := service.RemoveWhitelist(db, p); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

func putWhite(c *gin.Context) {
	var players []database.PlayerW
	if err := c.ShouldBindJSON(&players); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	if err := service.PutWhitelist(db, players); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}
