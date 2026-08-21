package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"palworld-panel/internal/database"
	"palworld-panel/service"
)

func listGuilds(c *gin.Context) {
	guilds, err := service.ListGuilds(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, guilds)
}

func getGuild(c *gin.Context) {
	uid := c.Param("admin_player_uid")
	g, err := service.GetGuild(db, uid)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "公会不存在"})
		return
	}
	c.JSON(http.StatusOK, g)
}

func putGuilds(c *gin.Context) {
	var guilds []database.Guild
	if err := c.ShouldBindJSON(&guilds); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	if err := service.PutGuilds(db, guilds); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}
