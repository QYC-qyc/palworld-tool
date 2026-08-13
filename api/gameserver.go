package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"paladmin/internal/gamesrv"
)

type gameServerAPI struct {
	mgr *gamesrv.Manager
}

func newGameServerAPI() (*gameServerAPI, error) {
	mgr, err := gamesrv.NewManager()
	if err != nil {
		return nil, err
	}
	return &gameServerAPI{mgr: mgr}, nil
}

func (g *gameServerAPI) status(c *gin.Context) {
	if !g.mgr.Available() {
		c.JSON(http.StatusOK, gin.H{"available": false, "message": "无法连接 Docker，请确认面板容器已挂载 /var/run/docker.sock"})
		return
	}
	st, err := g.mgr.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"available": true, "status": st})
}

func (g *gameServerAPI) install(c *gin.Context) {
	var cfg gamesrv.Config
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	msg, err := g.mgr.Install(cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: msg})
}

func (g *gameServerAPI) start(c *gin.Context) {
	if err := g.mgr.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

func (g *gameServerAPI) stop(c *gin.Context) {
	if err := g.mgr.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

func (g *gameServerAPI) restart(c *gin.Context) {
	if err := g.restartImpl(); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

// restartImpl 供其他模块（配置保存）调用
func (g *gameServerAPI) restartImpl() error {
	return g.mgr.Restart()
}

func (g *gameServerAPI) update(c *gin.Context) {
	var cfg gamesrv.Config
	_ = c.ShouldBindJSON(&cfg)
	msg, err := g.mgr.Update(cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: msg})
}

func (g *gameServerAPI) logs(c *gin.Context) {
	out, err := g.mgr.Logs(200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": out})
}
