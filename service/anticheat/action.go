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

// ActionExecutor 通过游戏官方 REST/RCON 执行处置动作（纯 Linux 原生，无需 Wine/PalDefender）。
type ActionExecutor struct {
	db  *bbolt.DB
	cfg *config.AnticheatConfig
}

func NewActionExecutor(db *bbolt.DB, cfg *config.AnticheatConfig) *ActionExecutor {
	return &ActionExecutor{db: db, cfg: cfg}
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
				if f.Nickname != "" {
					_ = tool.Broadcast(fmt.Sprintf("[反作弊] %s: %s", f.Nickname, msg))
				}
			}
		case ActionKick:
			if e.cfg.Punish.Kick {
				targetID := f.effectiveSteamID()
				err = tool.KickPlayer(targetID)
				logger.Warnf("踢出 %s (%s): %v", f.Nickname, f.Rule.Reason, err)
				_ = AddAudit(e.db, "anticheat", "kick", targetID, f.Title, resultStr(err))
			}
		case ActionBan:
			if e.cfg.Punish.Ban {
				if e.cfg.Punish.BackupBeforeBan {
					if _, berr := tool.Backup(); berr != nil {
						logger.Errorf("封禁前备份失败: %v", berr)
					}
				}
				targetID := f.effectiveSteamID()
				err = tool.BanPlayer(targetID)
				_ = service.AddBan(e.db, database.BanRecord{
					Type: database.BanUser, Identifier: targetID,
					Reason: f.Rule.Reason, Issuer: "anticheat",
				})
				logger.Warnf("封禁 %s (%s): %v", f.Nickname, f.Rule.Reason, err)
				_ = AddAudit(e.db, "anticheat", "ban", targetID, f.Title, resultStr(err))
				if e.cfg.Punish.Announce {
					_ = tool.Broadcast(fmt.Sprintf("[反作弊] 玩家 %s 因 %s 被封禁", f.Nickname, f.Rule.Reason))
				}
			}
		case ActionIPBan:
			if e.cfg.Punish.IPBan {
				targetID := f.effectiveSteamID()
				if p, perr := service.GetPlayer(e.db, f.PlayerUID); perr == nil && p.Ip != "" {
					_ = service.AddBan(e.db, database.BanRecord{
						Type: database.BanIP, Identifier: p.Ip,
						Reason: f.Rule.Reason, Issuer: "anticheat",
					})
					_ = tool.BanPlayer(targetID)
				}
				_ = AddAudit(e.db, "anticheat", "ipban", targetID, f.Title, "attempted")
			}
		}
	}
	return err
}

// effectiveSteamID 返回用于处置的 SteamID（优先 SteamID 字段，兼容 UserID）
func (f Finding) effectiveSteamID() string {
	if f.SteamID != "" {
		return f.SteamID
	}
	return f.UserID
}

func resultStr(err error) string {
	if err != nil {
		return "fail: " + err.Error()
	}
	return "success"
}
