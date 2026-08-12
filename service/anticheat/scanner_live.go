package anticheat

import (
	"fmt"
	"math"
	"time"

	"paladmin/internal/database"
)

// LiveScanner 在线行为扫描器，维护上一次轮询的玩家状态
type LiveScanner struct {
	rules   map[string]Rule
	prev    map[string]liveState
	first   bool
	joinCnt map[string]int // 短时间重连计数（简化）
}

type liveState struct {
	X, Y  float64
	Level int32
	At    time.Time
	IP    string
}

func NewLiveScanner(rules map[string]Rule) *LiveScanner {
	return &LiveScanner{rules: rules, prev: map[string]liveState{}, first: true, joinCnt: map[string]int{}}
}

// Scan 对当前在线玩家做行为检测
func (s *LiveScanner) Scan(players []database.OnlinePlayer, whitelist []database.PlayerW) []Finding {
	var findings []Finding
	current := map[string]liveState{}

	// L002 同 IP 多开
	if r, ok := s.rules["L002"]; ok && r.Enabled {
		ipCount := map[string]int{}
		for _, p := range players {
			if p.Ip != "" {
				ipCount[p.Ip]++
			}
		}
		maxAcc := toInt(r.Params["max_accounts"])
		if maxAcc <= 0 {
			maxAcc = 3
		}
		for _, p := range players {
			if p.Ip != "" && ipCount[p.Ip] > maxAcc && !isWhitelisted(p, whitelist) {
				findings = append(findings, Finding{
					Rule: r, PlayerUID: p.PlayerUid, SteamID: p.SteamId, Nickname: p.Nickname, Source: "internal",
					Title:  fmt.Sprintf("同IP多开(%s): %d 个账号", p.Ip, ipCount[p.Ip]),
					Detail: map[string]interface{}{"ip": p.Ip, "count": ipCount[p.Ip]},
				})
			}
		}
	}

	for _, p := range players {
		if isWhitelisted(p, whitelist) {
			current[p.PlayerUid] = liveState{p.LocationX, p.LocationY, p.Level, time.Now(), p.Ip}
			continue
		}
		now := time.Now()

		// L001 瞬移检测
		if r, ok := s.rules["L001"]; ok && r.Enabled && !s.first {
			if old, exists := s.prev[p.PlayerUid]; exists {
				maxSpeed := toFloat(r.Params["max_speed"])
				if maxSpeed <= 0 {
					maxSpeed = 2500
				}
				dist := math.Hypot(p.LocationX-old.X, p.LocationY-old.Y)
				dt := now.Sub(old.At).Seconds()
				if dt > 0 && dist/dt > maxSpeed && dist > 1000 {
					findings = append(findings, Finding{
						Rule: r, PlayerUID: p.PlayerUid, SteamID: p.SteamId, Nickname: p.Nickname, Source: "internal",
						Title:  fmt.Sprintf("移动速度异常: %.0f 单位/秒", dist/dt),
						Detail: map[string]interface{}{"speed": dist / dt, "from_x": old.X, "from_y": old.Y, "to_x": p.LocationX, "to_y": p.LocationY},
					})
				}
			}
		}

		// L004 等级突变
		if r, ok := s.rules["L004"]; ok && r.Enabled && !s.first {
			if old, exists := s.prev[p.PlayerUid]; exists && p.Level-old.Level > 20 {
				findings = append(findings, Finding{
					Rule: r, PlayerUID: p.PlayerUid, SteamID: p.SteamId, Nickname: p.Nickname, Source: "internal",
					Title: fmt.Sprintf("在线等级突变 %d -> %d", old.Level, p.Level),
					Detail: map[string]interface{}{"from": old.Level, "to": p.Level},
				})
			}
		}

		current[p.PlayerUid] = liveState{p.LocationX, p.LocationY, p.Level, now, p.Ip}
	}

	s.prev = current
	s.first = false
	return findings
}

// PlayerJoined 可由登录事件调用，用于 L003 重复登录检测
func (s *LiveScanner) CheckDuplicateLogin(uid string, online []database.OnlinePlayer) *Finding {
	r, ok := s.rules["L003"]
	if !ok || !r.Enabled {
		return nil
	}
	count := 0
	for _, p := range online {
		if p.PlayerUid == uid {
			count++
		}
	}
	if count > 1 {
		return &Finding{
			Rule: r, PlayerUID: uid, Source: "internal",
			Title:  fmt.Sprintf("账号重复登录(%d 处)", count),
			Detail: map[string]interface{}{"sessions": count},
		}
	}
	return nil
}

func isWhitelisted(p database.OnlinePlayer, whitelist []database.PlayerW) bool {
	for _, w := range whitelist {
		if (p.PlayerUid != "" && p.PlayerUid == w.PlayerUID) ||
			(p.SteamId != "" && p.SteamId == w.SteamID) ||
			(p.Nickname != "" && p.Nickname == w.Name) {
			return true
		}
	}
	return false
}
