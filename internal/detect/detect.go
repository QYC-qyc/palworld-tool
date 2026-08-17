// Package detect 提供轻量级外部反作弊检测。
// 不 hook 游戏进程，通过 REST API 轮询在线玩家和解析存档数据发现异常。
// 检测到作弊时，封禁优先通过 PalDefender REST API（写入 Banlist.json 并踢在线玩家），
// 未配置 PalDefender 时回退到官方 REST API。
package detect

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.etcd.io/bbolt"
	"paladmin/internal/database"
	"paladmin/internal/logger"
	"paladmin/internal/paldefender"
	"paladmin/internal/tool"
	"paladmin/service"
)

// Config 检测配置
type Config struct {
	Enabled          bool
	OnlineInterval   int  // 在线检测间隔秒
	DuplicateLogin   bool // 重复登录
	SameIPMulti      bool // 同IP多开
	LevelJump        bool // 等级突变
	MaxLevelJump     int  // 等级变化阈值
	SaveCheck        bool // 存档校验
	IllegalPalStats  bool // 帕鲁非法属性
	DuplicatePals    bool // 复制帕鲁
	IllegalItems     bool // 非法物品
	BanOnDetect      bool // 检测到自动封禁
	KickOnDetect     bool // 检测到自动踢出
}

var (
	lastSnapshot = map[string]*PlayerSnap{}
	snapMu       sync.Mutex
)

// PlayerSnap 在线玩家快照
type PlayerSnap struct {
	SteamID string
	Level   int32
	IP      string
	Time    time.Time
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Enabled:         true,
		OnlineInterval:  10,
		DuplicateLogin:  true,
		SameIPMulti:     true,
		LevelJump:       true,
		MaxLevelJump:    10,
		SaveCheck:       true,
		IllegalPalStats: true,
		DuplicatePals:   true,
		IllegalItems:    true,
		BanOnDetect:     false,
		KickOnDetect:    true,
	}
}

// RunOnlineCheck 执行一次在线检测
func RunOnlineCheck(db *bbolt.DB, cfg *Config) {
	if cfg == nil || !cfg.Enabled {
		return
	}
	online, err := tool.ShowPlayers()
	if err != nil {
		logger.Debugf("在线检测：获取玩家失败: %v", err)
		return
	}

	now := time.Now()
	steamIDSet := map[string]bool{}
	ipCount := map[string]int{}
	ipToPlayers := map[string][]string{}

	for _, p := range online {
		steamIDSet[p.SteamId] = true
		if p.Ip != "" {
			ipCount[p.Ip]++
			ipToPlayers[p.Ip] = append(ipToPlayers[p.Ip], p.Nickname)
		}

		// 等级突变检测
		if cfg.LevelJump && cfg.MaxLevelJump > 0 {
			snapMu.Lock()
			old, exists := lastSnapshot[p.PlayerUid]
			snapMu.Unlock()

			if exists && p.Level > 0 && old.Level > 0 {
				jump := int(p.Level - old.Level)
				if jump > cfg.MaxLevelJump && now.Sub(old.Time) < 10*time.Minute {
					act(db, cfg, p.SteamId, p.Nickname,
						fmt.Sprintf("等级在 %v 内从 %d 升至 %d（+%d）",
							now.Sub(old.Time).Round(time.Second), old.Level, p.Level, jump))
				}
			}
			snapMu.Lock()
			lastSnapshot[p.PlayerUid] = &PlayerSnap{
				SteamID: p.SteamId, Level: p.Level, IP: p.Ip, Time: now,
			}
			snapMu.Unlock()
		}
	}

	// 同IP多开检测
	if cfg.SameIPMulti {
		for ip, count := range ipCount {
			if count > 1 {
				names := strings.Join(ipToPlayers[ip], ", ")
				for _, p := range online {
					if p.Ip == ip {
						act(db, cfg, p.SteamId, p.Nickname,
							fmt.Sprintf("同IP(%s)多开：%s", ip, names))
					}
				}
			}
		}
	}

	// 清理离线玩家快照
	snapMu.Lock()
	for uid := range lastSnapshot {
		found := false
		for _, p := range online {
			if p.PlayerUid == uid {
				found = true
				break
			}
		}
		if !found {
			delete(lastSnapshot, uid)
		}
	}
	snapMu.Unlock()
}

// RunSaveCheck 执行存档数据校验（在存档解析完成后由 API 触发）
func RunSaveCheck(db *bbolt.DB, cfg *Config, players []database.Player) {
	if cfg == nil || !cfg.Enabled || !cfg.SaveCheck {
		return
	}

	for _, p := range players {
		// 帕鲁属性校验
		if cfg.IllegalPalStats {
			for _, pal := range p.Pals {
				if pal == nil {
					continue
				}
				var issues []string
				if pal.Level < 1 || pal.Level > 200 {
					issues = append(issues, fmt.Sprintf("等级 %d 异常", pal.Level))
				}
				if pal.Rank < 0 || pal.Rank > 5 {
					issues = append(issues, fmt.Sprintf("强化等级 %d 异常", pal.Rank))
				}
				if pal.Hp < 0 || pal.MaxHp > 999999 {
					issues = append(issues, fmt.Sprintf("HP %d/%d 异常", pal.Hp, pal.MaxHp))
				}
				// 天赋范围检查 (0-100)
				if pal.TalentHP < 0 || pal.TalentHP > 100 ||
					pal.TalentMelee < 0 || pal.TalentMelee > 100 ||
					pal.TalentRanged < 0 || pal.TalentRanged > 100 ||
					pal.TalentDefense < 0 || pal.TalentDefense > 100 {
					issues = append(issues, "天赋值超出 0-100 范围")
				}
				if len(issues) > 0 {
					act(db, cfg, p.SteamId, p.Nickname,
						fmt.Sprintf("帕鲁 %s(%s) 异常：%s",
							pal.Type, pal.InstanceID[:8], strings.Join(issues, "；")))
				}
			}
		}

		// 复制帕鲁检测（同 InstanceID 出现多次）
		if cfg.DuplicatePals {
			instCount := map[string]int{}
			for _, pal := range p.Pals {
				if pal != nil && pal.InstanceID != "" {
					instCount[pal.InstanceID]++
				}
			}
			for inst, n := range instCount {
				if n > 1 {
					act(db, cfg, p.SteamId, p.Nickname,
						fmt.Sprintf("疑似复制帕鲁：%s 出现 %d 次", inst[:8], n))
				}
			}
		}

		// 非法物品检测（堆叠数异常）
		if cfg.IllegalItems && p.Items != nil {
			containers := [][]*database.Item{
				p.Items.CommonContainerId,
				p.Items.DropSlotContainerId,
				p.Items.EssentialContainerId,
				p.Items.WeaponLoadOutContainerId,
				p.Items.PlayerEquipArmorContainerId,
				p.Items.FoodEquipContainerId,
			}
			for _, container := range containers {
				for _, item := range container {
					if item != nil && item.StackCount > 9999 {
						act(db, cfg, p.SteamId, p.Nickname,
							fmt.Sprintf("物品 %s 堆叠数 %d 异常", item.ItemId, item.StackCount))
					}
				}
			}
		}
	}
}

// act 执行处置动作。封禁优先通过 PalDefender（写入 Banlist.json 并强制踢在线玩家），
// 未配置 PalDefender 时回退到官方 REST API。
func act(db *bbolt.DB, cfg *Config, steamID, nickname, reason string) {
	logger.Warnf("[检测] %s(%s): %s", nickname, steamID, reason)
	if db == nil || steamID == "" {
		return
	}

	// 封禁（优先 PalDefender）
	banned := false
	if cfg.BanOnDetect {
		banned = banByPalDefender(db, steamID, reason)
		if !banned {
			if err := tool.BanPlayer(steamID); err != nil {
				logger.Warnf("封禁失败(官方REST): %v", err)
			} else {
				banned = true
			}
		}
		if banned {
			_ = service.AddBan(db, database.BanRecord{
				Type: database.BanUser, Identifier: steamID,
				Reason: reason, Issuer: "detect",
			})
			logger.Infof("已封禁 %s: %s", nickname, reason)
		}
	}

	// 踢出：未封禁或封禁未把玩家踢下线时执行
	if cfg.KickOnDetect && !banned {
		if err := tool.KickPlayer(steamID); err != nil {
			logger.Warnf("踢出失败: %v", err)
		} else {
			logger.Infof("已踢出 %s: %s", nickname, reason)
		}
	}
}

// banByPalDefender 尝试通过 PalDefender REST API 封禁玩家。
// 成功返回 true（PD 封禁会同时踢在线玩家）；未配置或失败返回 false 由调用方回退。
func banByPalDefender(db *bbolt.DB, steamID, reason string) bool {
	client, err := paldefender.Load(db)
	if err != nil {
		if !errors.Is(err, paldefender.ErrNotConfigured) {
			logger.Debugf("PalDefender 加载失败: %v", err)
		}
		return false
	}
	// PD 接受带平台前缀的 UserId（steam_xxx）
	if _, err := client.Ban("steam_"+steamID, reason, false); err != nil {
		logger.Warnf("PalDefender 封禁失败: %v", err)
		return false
	}
	return true
}
