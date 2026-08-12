package anticheat

import (
	"time"

	"paladmin/internal/database"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarn     Severity = "warn"
	SeverityCritical Severity = "critical"
)

type Action string

const (
	ActionWarn  Action = "warn"
	ActionKick  Action = "kick"
	ActionBan   Action = "ban"
	ActionIPBan Action = "ipban"
)

// Rule 规则定义
type Rule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Category    string                 `json:"category"` // save/live/integrated
	Enabled     bool                   `json:"enabled"`
	Severity    Severity               `json:"severity"`
	Actions     []Action               `json:"actions"`
	Reason      string                 `json:"reason"`
	Params      map[string]interface{} `json:"params"`
	Description string                 `json:"description"`
}

// Finding 检测结果
type Finding struct {
	Rule      Rule
	UserID    string // steam_xxx / gdk_xxx
	PlayerUID string
	SteamID   string
	Nickname  string
	PalInstID string
	Title     string
	Detail    map[string]interface{}
	Evidence  []byte
	Source    string
}

// 默认规则集
func DefaultRules() []Rule {
	return []Rule{
		{ID: "S001", Name: "帕鲁属性越界", Category: "save", Enabled: true, Severity: SeverityCritical,
			Actions: []Action{ActionWarn, ActionBan}, Reason: "非法帕鲁属性",
			Description: "HP/攻击/防御/工作速度/等级/rank 超过合法上限"},
		{ID: "S002", Name: "非法天赋/灵魂", Category: "save", Enabled: true, Severity: SeverityCritical,
			Actions: []Action{ActionWarn, ActionBan}, Reason: "非法天赋或帕鲁灵魂",
			Description: "IV 不在 0-100，或 PalSouls 越界",
			Params: map[string]interface{}{"max_iv": 100, "max_soul": 20}},
		{ID: "S003", Name: "非法词条", Category: "save", Enabled: true, Severity: SeverityWarn,
			Actions: []Action{ActionKick}, Reason: "非法被动词条",
			Description: "被动技能 ID 非法或数量过多"},
		{ID: "S004", Name: "非法Boss/塔主帕鲁", Category: "save", Enabled: true, Severity: SeverityCritical,
			Actions: []Action{ActionBan}, Reason: "持有Boss/塔主帕鲁"},
		{ID: "S005", Name: "物品堆叠越界", Category: "save", Enabled: true, Severity: SeverityWarn,
			Actions: []Action{ActionWarn}, Reason: "物品堆叠数异常",
			Params: map[string]interface{}{"default_max_stack": 9999}},
		{ID: "S006", Name: "非法物品", Category: "save", Enabled: true, Severity: SeverityCritical,
			Actions: []Action{ActionBan}, Reason: "持有非法物品"},
		{ID: "S007", Name: "复制帕鲁", Category: "save", Enabled: true, Severity: SeverityCritical,
			Actions: []Action{ActionBan}, Reason: "检测到复制帕鲁",
			Description: "相同 InstanceID 出现在多个玩家身上"},
		{ID: "S008", Name: "玩家属性越界", Category: "save", Enabled: true, Severity: SeverityCritical,
			Actions: []Action{ActionBan}, Reason: "玩家属性异常",
			Params: map[string]interface{}{"max_level": 100}},
		{ID: "S009", Name: "经验异常", Category: "save", Enabled: true, Severity: SeverityWarn,
			Actions: []Action{ActionWarn}, Reason: "经验值异常"},
		{ID: "S010", Name: "进度异常增长", Category: "save", Enabled: false, Severity: SeverityWarn,
			Actions: []Action{ActionKick}, Reason: "短时间进度异常增长"},
		{ID: "S011", Name: "非法据点", Category: "save", Enabled: false, Severity: SeverityInfo,
			Actions: []Action{ActionWarn}, Reason: "据点异常"},
		{ID: "S012", Name: "资源异常", Category: "save", Enabled: false, Severity: SeverityWarn,
			Actions: []Action{ActionWarn}, Reason: "资源/金钱异常"},
		{ID: "L001", Name: "速度异常/瞬移", Category: "live", Enabled: true, Severity: SeverityWarn,
			Actions: []Action{ActionWarn, ActionKick}, Reason: "移动速度异常",
			Params: map[string]interface{}{"max_speed": 2500, "teleport_whitelist": []interface{}{}}},
		{ID: "L002", Name: "同IP多开", Category: "live", Enabled: false, Severity: SeverityInfo,
			Actions: []Action{ActionWarn}, Reason: "同一IP多账号在线",
			Params: map[string]interface{}{"max_accounts": 3}},
		{ID: "L003", Name: "重复登录", Category: "live", Enabled: true, Severity: SeverityWarn,
			Actions: []Action{ActionKick}, Reason: "同一账号重复登录"},
		{ID: "L004", Name: "在线等级突变", Category: "live", Enabled: false, Severity: SeverityWarn,
			Actions: []Action{ActionKick}, Reason: "在线等级异常跳变"},
		{ID: "L005", Name: "非白名单在线", Category: "live", Enabled: false, Severity: SeverityWarn,
			Actions: []Action{ActionKick}, Reason: "不在白名单中"},
	}
}

// alertFromFinding 将 Finding 转为待持久化的 Alert
func alertFromFinding(f Finding) database.Alert {
	detail := ""
	if f.Detail != nil {
		detail = mapToJSON(f.Detail)
	}
	return database.Alert{
		RuleID:      f.Rule.ID,
		Severity:    string(f.Rule.Severity),
		PlayerUID:   f.PlayerUID,
		SteamID:     f.SteamID,
		Nickname:    f.Nickname,
		PalInstID:   f.PalInstID,
		Title:       f.Title,
		Detail:      detail,
		Status:      "open",
		ActionTaken: "",
		Source:      f.Source,
		CreatedAt:   time.Now(),
	}
}
