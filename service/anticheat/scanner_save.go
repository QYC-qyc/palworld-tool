package anticheat

import (
	"fmt"

	"paladmin/internal/database"
)

// SaveScanner 存档扫描器
type SaveScanner struct {
	rules map[string]Rule
	data  *GameData
}

func NewSaveScanner(rules map[string]Rule, data *GameData) *SaveScanner {
	return &SaveScanner{rules: rules, data: data}
}

// Scan 对全体玩家执行存档类规则检测
func (s *SaveScanner) Scan(players []database.Player) []Finding {
	var findings []Finding

	// S007 复制帕鲁检测：instanceID -> 拥有者
	ownerByInstance := make(map[string]string)
	// 同 IV/技能/等级指纹 -> 拥有者（辅助复制检测）
	type palFP struct {
		palType             string
		level, rank         int32
		tHP, tMelee, tRange int32
	}
	fpByOwner := make(map[palFP]string)

	for _, p := range players {
		for _, pal := range p.Pals {
			if pal == nil {
				continue
			}
			if r, ok := s.rules["S001"]; ok && r.Enabled {
				if f, hit := s.checkPalStats(r, p, pal); hit {
					findings = append(findings, f)
				}
			}
			if r, ok := s.rules["S002"]; ok && r.Enabled {
				if f, hit := s.checkTalent(r, p, pal); hit {
					findings = append(findings, f)
				}
			}
			if r, ok := s.rules["S003"]; ok && r.Enabled {
				if f, hit := s.checkPassives(r, p, pal); hit {
					findings = append(findings, f)
				}
			}
			if r, ok := s.rules["S004"]; ok && r.Enabled {
				if pal.IsBoss || pal.IsTower || s.data.IsBossPalID(pal.Type) {
					findings = append(findings, newPalFinding(r, p, pal,
						fmt.Sprintf("玩家持有非法 Boss/塔主帕鲁 %s", pal.Type)))
				}
			}

			// S007 复制检测：InstanceID
			if pal.InstanceID != "" {
				if owner, exists := ownerByInstance[pal.InstanceID]; exists && owner != p.PlayerUid {
					if r, ok := s.rules["S007"]; ok && r.Enabled {
						findings = append(findings, Finding{
							Rule: r, PlayerUID: p.PlayerUid, SteamID: p.SteamId, Nickname: p.Nickname,
							PalInstID: pal.InstanceID, Source: "internal",
							Title: fmt.Sprintf("复制帕鲁 %s（InstanceID 同时属于 %s 与 %s）", pal.Type, owner, p.PlayerUid),
							Detail: map[string]interface{}{"instance_id": pal.InstanceID, "pal_type": pal.Type, "other_owner": owner},
						})
					}
				} else {
					ownerByInstance[pal.InstanceID] = p.PlayerUid
				}
			}
			// 指纹复制检测（InstanceID 缺失时的兜底）
			fp := palFP{pal.Type, pal.Level, pal.Rank, pal.TalentHP, pal.TalentMelee, pal.TalentRanged}
			if owner, exists := fpByOwner[fp]; exists && owner != p.PlayerUid && pal.Level > 10 {
				if r, ok := s.rules["S007"]; ok && r.Enabled {
					findings = append(findings, Finding{
						Rule: r, PlayerUID: p.PlayerUid, SteamID: p.SteamId, Nickname: p.Nickname,
						Source: "internal",
						Title:  fmt.Sprintf("疑似复制帕鲁 %s（与 %s 特征完全一致）", pal.Type, owner),
						Detail: map[string]interface{}{"pal_type": pal.Type, "level": pal.Level, "rank": pal.Rank},
					})
				}
			} else {
				fpByOwner[fp] = p.PlayerUid
			}
		}

		if r, ok := s.rules["S008"]; ok && r.Enabled {
			if f, hit := s.checkPlayerStats(r, p); hit {
				findings = append(findings, f)
			}
		}
		if r, ok := s.rules["S005"]; ok && r.Enabled {
			findings = append(findings, s.checkItems(r, p)...)
		}
		if r, ok := s.rules["S006"]; ok && r.Enabled {
			findings = append(findings, s.checkIllegalItems(r, p)...)
		}
	}
	return findings
}

func (s *SaveScanner) checkPalStats(r Rule, p database.Player, pal *database.Pal) (Finding, bool) {
	pl := s.data.Limits.Pal
	if pal.Level <= 0 || pal.Level > int32(pl.MaxLevel) {
		return newPalFinding(r, p, pal, fmt.Sprintf("%s 等级越界: %d", pal.Type, pal.Level)), true
	}
	if pal.Rank < 0 || pal.Rank > int32(pl.MaxRank) {
		return newPalFinding(r, p, pal, fmt.Sprintf("%s 阶级越界: %d", pal.Type, pal.Rank)), true
	}
	if pal.MaxHp > int64(pl.MaxHP) || pal.Melee > int32(pl.MaxMelee) || pal.Ranged > int32(pl.MaxRanged) ||
		pal.Defense > int32(pl.MaxDefense) || pal.Workspeed > int32(pl.MaxWorkspeed) {
		return newPalFinding(r, p, pal, fmt.Sprintf("%s 属性越界: hp=%d atk=%d/%d def=%d work=%d",
			pal.Type, pal.MaxHp, pal.Melee, pal.Ranged, pal.Defense, pal.Workspeed)), true
	}
	// 类型合法性
	if pal.Type != "Unknow" && !s.data.IsLegalPalID(pal.Type) && !s.data.IsBossPalID(pal.Type) {
		return newPalFinding(r, p, pal, fmt.Sprintf("非法帕鲁类型 ID: %s", pal.Type)), true
	}
	return Finding{}, false
}

func (s *SaveScanner) checkTalent(r Rule, p database.Player, pal *database.Pal) (Finding, bool) {
	pl := s.data.Limits.Pal
	maxIV := pl.MaxTalent
	if maxIV <= 0 {
		maxIV = 100
	}
	for _, v := range []int32{pal.TalentHP, pal.TalentMelee, pal.TalentRanged, pal.TalentDefense} {
		if v < 0 || v > int32(maxIV) {
			return newPalFinding(r, p, pal, fmt.Sprintf("%s 天赋/IV 越界: %d", pal.Type, v)), true
		}
	}
	maxSoul := pl.MaxSoul
	for _, v := range []int32{pal.SoulHP, pal.SoulATK, pal.SoulDEF, pal.SoulCS} {
		if v < 0 || v > int32(maxSoul) {
			return newPalFinding(r, p, pal, fmt.Sprintf("%s 灵魂强化越界: %d", pal.Type, v)), true
		}
	}
	return Finding{}, false
}

func (s *SaveScanner) checkPassives(r Rule, p database.Player, pal *database.Pal) (Finding, bool) {
	pl := s.data.Limits.Pal
	maxPassives := pl.MaxPassives
	if maxPassives <= 0 {
		maxPassives = 4
	}
	if len(pal.Skills) > maxPassives {
		return newPalFinding(r, p, pal, fmt.Sprintf("%s 词条数量异常: %d", pal.Type, len(pal.Skills))), true
	}
	for _, sk := range pal.Skills {
		if sk == "" {
			return newPalFinding(r, p, pal, fmt.Sprintf("%s 含空词条", pal.Type)), true
		}
	}
	return Finding{}, false
}

func (s *SaveScanner) checkPlayerStats(r Rule, p database.Player) (Finding, bool) {
	pl := s.data.Limits.Player
	if p.Level > int32(pl.MaxLevel) || p.Level < 0 {
		return Finding{
			Rule: r, PlayerUID: p.PlayerUid, SteamID: p.SteamId, Nickname: p.Nickname, Source: "internal",
			Title:  fmt.Sprintf("玩家等级越界: %d", p.Level),
			Detail: map[string]interface{}{"level": p.Level},
		}, true
	}
	if p.MaxHp > pl.MaxHP {
		return Finding{
			Rule: r, PlayerUID: p.PlayerUid, SteamID: p.SteamId, Nickname: p.Nickname, Source: "internal",
			Title:  fmt.Sprintf("玩家 HP 越界: %d", p.MaxHp),
			Detail: map[string]interface{}{"max_hp": p.MaxHp},
		}, true
	}
	return Finding{}, false
}

func (s *SaveScanner) checkItems(r Rule, p database.Player) []Finding {
	var out []Finding
	if p.Items == nil {
		return out
	}
	containers := map[string][]*database.Item{
		"普通物品栏": p.Items.CommonContainerId,
		"掉落栏":    p.Items.DropSlotContainerId,
		"重要物品栏": p.Items.EssentialContainerId,
		"食物装备栏": p.Items.FoodEquipContainerId,
		"防具栏":    p.Items.PlayerEquipArmorContainerId,
		"武器栏":    p.Items.WeaponLoadOutContainerId,
	}
	for cname, items := range containers {
		for _, it := range items {
			if it == nil {
				continue
			}
			max := s.data.MaxStackFor(it.ItemId)
			if it.StackCount > int32(max) {
				out = append(out, Finding{
					Rule: r, PlayerUID: p.PlayerUid, SteamID: p.SteamId, Nickname: p.Nickname, Source: "internal",
					Title: fmt.Sprintf("物品堆叠越界 %s: %d > %d (%s)", it.ItemId, it.StackCount, max, cname),
					Detail: map[string]interface{}{
						"item_id": it.ItemId, "stack": it.StackCount, "max": max, "container": cname,
					},
				})
			}
		}
	}
	return out
}

func (s *SaveScanner) checkIllegalItems(r Rule, p database.Player) []Finding {
	var out []Finding
	if p.Items == nil {
		return out
	}
	all := [][]*database.Item{
		p.Items.CommonContainerId, p.Items.DropSlotContainerId, p.Items.EssentialContainerId,
		p.Items.FoodEquipContainerId, p.Items.PlayerEquipArmorContainerId, p.Items.WeaponLoadOutContainerId,
	}
	for _, items := range all {
		for _, it := range items {
			if it == nil || it.ItemId == "" {
				continue
			}
			if !s.data.IsLegalItem(it.ItemId) {
				out = append(out, Finding{
					Rule: r, PlayerUID: p.PlayerUid, SteamID: p.SteamId, Nickname: p.Nickname, Source: "internal",
					Title:  fmt.Sprintf("持有非法物品: %s", it.ItemId),
					Detail: map[string]interface{}{"item_id": it.ItemId, "stack": it.StackCount},
				})
			}
		}
	}
	return out
}

func newPalFinding(r Rule, p database.Player, pal *database.Pal, title string) Finding {
	return Finding{
		Rule: r, PlayerUID: p.PlayerUid, SteamID: p.SteamId, Nickname: p.Nickname,
		PalInstID: pal.InstanceID, Source: "internal", Title: title,
		Detail: map[string]interface{}{
			"pal_type": pal.Type, "level": pal.Level, "max_hp": pal.MaxHp,
			"melee": pal.Melee, "ranged": pal.Ranged, "defense": pal.Defense,
			"talents": []int32{pal.TalentHP, pal.TalentMelee, pal.TalentRanged, pal.TalentDefense},
			"souls":   []int32{pal.SoulHP, pal.SoulATK, pal.SoulDEF, pal.SoulCS},
			"skills":  pal.Skills,
		},
	}
}
