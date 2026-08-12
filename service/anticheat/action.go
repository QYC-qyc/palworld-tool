package anticheat

import (
	"fmt"

	"go.etcd.io/bbolt"
	"paladmin/internal/config"
	"paladmin/internal/database"
	"paladmin/internal/logger"
	"paladmin/internal/tool"
	"paladmin/service"
)

// ActionExecutor 执行处置动作。集成模式下优先用 PalDefender，否则用官方 REST/RCON。
type ActionExecutor struct {
	db  *bbolt.DB
	cfg *config.AnticheatConfig
	pd  *tool.PalDefenderClient
}

func NewActionExecutor(db *bbolt.DB, cfg *config.AnticheatConfig) *ActionExecutor {
	e := &ActionExecutor{db: db, cfg: cfg}
	if cfg.Mode == "integrated" {
		e.pd = tool.NewPalDefender()
		if !e.pd.Available() {
			logger.Warn("反作弊模式为 integrated，但 PalDefender 未配置完整，回退到官方 REST")
			e.pd = nil
		}
	}
	return e
}

// Execute 对一个 Finding 执行其规则的全部动作
func (e *ActionExecutor) Execute(f Finding) error {
	var err error
	for _, act := range f.Rule.Actions {
		switch act {
		case ActionWarn:
			if e.cfg.Punish.Warn {
				msg := "检测到异常行为"
				if e.cfg.Punish.WarnWithReason && f.Rule.Reason != "" {
					msg = fmt.Sprintf("警告: %s", f.Rule.Reason)
				}
				// 优先私聊玩家，否则全服通告
				if e.pd != nil && f.UserID != "" {
					_ = e.pd.SendMessage([]string{f.UserID}, "PlayerLogImportant", msg)
				} else if f.Nickname != "" {
					_ = tool.Broadcast(fmt.Sprintf("[反作弊] %s: %s", f.Nickname, msg))
				}
			}
		case ActionKick:
			if e.cfg.Punish.Kick {
				if e.pd != nil {
					err = e.pd.Kick(f.UserID, f.Rule.Reason)
				} else {
					err = tool.KickPlayer(f.UserID)
				}
				logger.Warnf("踢出 %s (%s): %v", f.Nickname, f.Rule.Reason, err)
				_ = AddAudit(e.db, "anticheat", "kick", f.UserID, f.Title, resultStr(err))
			}
		case ActionBan:
			if e.cfg.Punish.Ban {
				if e.cfg.Punish.BackupBeforeBan {
					if _, berr := tool.Backup(); berr != nil {
						logger.Errorf("封禁前备份失败: %v", berr)
					}
				}
				if e.pd != nil {
					err = e.pd.Ban(f.UserID, f.Rule.Reason, false)
				} else {
					err = tool.BanPlayer(f.UserID)
				}
				_ = service.AddBan(e.db, database.BanRecord{
					Type: database.BanUser, Identifier: f.UserID,
					Reason: f.Rule.Reason, Issuer: "anticheat",
				})
				logger.Warnf("封禁 %s (%s): %v", f.Nickname, f.Rule.Reason, err)
				_ = AddAudit(e.db, "anticheat", "ban", f.UserID, f.Title, resultStr(err))
				if e.cfg.Punish.Announce {
					announce := fmt.Sprintf("[反作弊] 玩家 %s 因 %s 被封禁", f.Nickname, f.Rule.Reason)
					if e.pd != nil {
						_ = e.pd.Broadcast(announce)
					} else {
						_ = tool.Broadcast(announce)
					}
				}
			}
		case ActionIPBan:
			if e.cfg.Punish.IPBan {
				if p, perr := service.GetPlayer(e.db, f.PlayerUID); perr == nil && p.Ip != "" {
					_ = service.AddBan(e.db, database.BanRecord{
						Type: database.BanIP, Identifier: p.Ip,
						Reason: f.Rule.Reason, Issuer: "anticheat",
					})
					if e.pd != nil {
						_ = e.pd.BanIP(p.Ip, f.Rule.Reason, f.UserID)
					} else {
						_ = tool.BanPlayer(f.UserID)
					}
				}
				_ = AddAudit(e.db, "anticheat", "ipban", f.UserID, f.Title, "attempted")
			}
		}
	}
	return err
}

func resultStr(err error) string {
	if err != nil {
		return "fail: " + err.Error()
	}
	return "success"
}
