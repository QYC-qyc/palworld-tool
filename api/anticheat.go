package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"paladmin/internal/database"
	"paladmin/service"
	"paladmin/service/anticheat"
)

func listAlerts(c *gin.Context) {
	status := c.Query("status")
	playerUID := c.Query("player_uid")
	limit := queryInt(c, "limit", 100)
	offset := queryInt(c, "offset", 0)
	alerts, total, err := anticheat.ListAlerts(db, status, playerUID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"alerts": alerts, "total": total})
}

func getAlert(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的 id"})
		return
	}
	a, err := anticheat.GetAlert(db, id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, a)
}

type alertActionRequest struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

func alertAction(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的 id"})
		return
	}
	var req alertActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	if err := anticheat.UpdateAlertStatus(db, id, req.Status, req.Note); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	_ = anticheat.AddAudit(db, "web", "alert_"+req.Status, strconv.FormatUint(id, 10), req.Note, "success")
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

func listRules(c *gin.Context) {
	if engine == nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	c.JSON(http.StatusOK, engine.Rules())
}

type ruleUpdate struct {
	Enabled  *bool    `json:"enabled"`
	Severity string   `json:"severity"`
	Actions  []string `json:"actions"`
}

func updateRule(c *gin.Context) {
	id := c.Param("id")
	var req ruleUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	if engine != nil {
		engine.UpdateRule(id, req.Enabled, req.Severity, req.Actions)
	}
	_ = anticheat.AddAudit(db, "web", "rule_update", id, "更新反作弊规则", "success")
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

func runScan(c *gin.Context) {
	players, err := service.ListPlayers(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	full := make([]database.Player, 0, len(players))
	for _, tp := range players {
		if p, err := service.GetPlayer(db, tp.PlayerUid); err == nil {
			full = append(full, p)
		}
	}
	go func() {
		if engine != nil {
			engine.ScanSave(full)
		}
	}()
	c.JSON(http.StatusOK, gin.H{"scanned": len(full)})
}

func acStats(c *gin.Context) {
	alerts, _, err := anticheat.ListAlerts(db, "", "", 10000, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	bySeverity := map[string]int{}
	byStatus := map[string]int{}
	for _, a := range alerts {
		bySeverity[a.Severity]++
		byStatus[a.Status]++
	}
	bans, _ := service.ListBans(db, true)
	c.JSON(http.StatusOK, gin.H{
		"total_alerts": len(alerts),
		"by_severity":  bySeverity,
		"by_status":    byStatus,
		"active_bans":  len(bans),
	})
}

func listAudit(c *gin.Context) {
	limit := queryInt(c, "limit", 200)
	audits, err := anticheat.ListAudit(db, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, audits)
}

func reloadAC(c *gin.Context) {
	// 重新加载配置在当前进程中需重启，这里仅触发一次全量扫描
	go func() {
		if engine != nil {
			if players, err := service.ListPlayers(db); err == nil {
				full := make([]database.Player, 0)
				for _, tp := range players {
					if p, err := service.GetPlayer(db, tp.PlayerUid); err == nil {
						full = append(full, p)
					}
				}
				engine.ScanSave(full)
			}
		}
	}()
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}
