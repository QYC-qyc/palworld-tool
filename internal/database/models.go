package database

import "time"

// Pal 帕鲁数据，扩展反作弊所需字段
type Pal struct {
	InstanceID string   `json:"instance_id"`
	Level      int32    `json:"level"`
	Exp        int64    `json:"exp"`
	Hp         int64    `json:"hp"`
	MaxHp      int64    `json:"max_hp"`
	Type       string   `json:"type"`
	Gender     string   `json:"gender"`
	IsLucky    bool     `json:"is_lucky"`
	IsBoss     bool     `json:"is_boss"`
	IsTower    bool     `json:"is_tower"`
	Workspeed  int32    `json:"workspeed"`
	Melee      int32    `json:"melee"`
	Ranged     int32    `json:"ranged"`
	Defense    int32    `json:"defense"`
	Rank       int32    `json:"rank"`
	Skills     []string `json:"skills"`

	// 反作弊扩展
	TalentHP      int32 `json:"talent_hp"`       // 天赋/IV 0-100
	TalentMelee   int32 `json:"talent_melee"`
	TalentRanged  int32 `json:"talent_ranged"`
	TalentDefense int32 `json:"talent_defense"`
	SoulHP        int32 `json:"soul_hp"`         // 帕鲁灵魂强化
	SoulATK       int32 `json:"soul_atk"`
	SoulDEF       int32 `json:"soul_def"`
	SoulCS        int32 `json:"soul_cs"`
}

// OnlinePlayer 在线玩家实时数据（来自官方 REST）
type OnlinePlayer struct {
	PlayerUid  string    `json:"player_uid"`
	SteamId    string    `json:"steam_id"`
	Nickname   string    `json:"nickname"`
	Ip         string    `json:"ip"`
	Ping       float64   `json:"ping"`
	LocationX  float64   `json:"location_x"`
	LocationY  float64   `json:"location_y"`
	Level      int32     `json:"level"`
	LastOnline time.Time `json:"last_online"`
}

// GuildPlayer 公会成员
type GuildPlayer struct {
	PlayerUid string `json:"player_uid"`
	Nickname  string `json:"nickname"`
}

// TersePlayer 精简玩家
type TersePlayer struct {
	PlayerUid      string           `json:"player_uid"`
	Nickname       string           `json:"nickname"`
	PlatformID     string           `json:"platform_id,omitempty"`
	Level          int32            `json:"level"`
	Exp            int64            `json:"exp"`
	Hp             int64            `json:"hp"`
	MaxHp          int64            `json:"max_hp"`
	ShieldHp       int64            `json:"shield_hp"`
	ShieldMaxHp    int64            `json:"shield_max_hp"`
	MaxStatusPoint int32            `json:"max_status_point"`
	StatusPoint    map[string]int32 `json:"status_point"`
	FullStomach    float64          `json:"full_stomach"`
	SaveLastOnline string           `json:"save_last_online"`
	OnlinePlayer
}

// Player 完整玩家（含帕鲁与物品）
type Player struct {
	TersePlayer
	Pals  []*Pal `json:"pals"`
	Items *Items `json:"items"`
}

// Guild 公会
// BaseCamp 公会据点（坐标来自 Level.sav 的 BaseCampSaveData）
type BaseCamp struct {
	Id        string  `json:"id"`
	Area      float64 `json:"area"`
	LocationX float64 `json:"location_x"`
	LocationY float64 `json:"location_y"`
}

type Guild struct {
	Name           string         `json:"name"`
	BaseCampLevel  int32          `json:"base_camp_level"`
	AdminPlayerUid string         `json:"admin_player_uid"`
	Players        []*GuildPlayer `json:"players"`
	BaseIds        []string       `json:"base_ids"`
	BaseCamp       []BaseCamp     `json:"base_camp"`
}

// PlayerW 白名单玩家
type PlayerW struct {
	Name      string `json:"name"`
	SteamID   string `json:"steam_id"`
	PlayerUID string `json:"player_uid"`
}

// Items 物品容器集合
type Items struct {
	CommonContainerId           []*Item `json:"CommonContainerId"`
	DropSlotContainerId         []*Item `json:"DropSlotContainerId"`
	EssentialContainerId        []*Item `json:"EssentialContainerId"`
	FoodEquipContainerId        []*Item `json:"FoodEquipContainerId"`
	PlayerEquipArmorContainerId []*Item `json:"PlayerEquipArmorContainerId"`
	WeaponLoadOutContainerId    []*Item `json:"WeaponLoadOutContainerId"`
}

// Item 单物品
type Item struct {
	SlotIndex  int32  `json:"SlotIndex"`
	ItemId     string `json:"ItemId"`
	StackCount int32  `json:"StackCount"`
}

// Backup 备份元数据
type Backup struct {
	BackupId string    `json:"backup_id"`
	SaveTime time.Time `json:"save_time"`
	Path     string    `json:"path"`
}

// BanType 封禁类型
type BanType string

const (
	BanUser BanType = "user"
	BanIP   BanType = "ip"
)

// BanRecord 封禁记录（兼容 PalDefender Banlist.json）
type BanRecord struct {
	Type       BanType   `json:"type"`
	Identifier string    `json:"identifier"` // UserId 或 IP
	Reason     string    `json:"reason"`
	Issuer     string    `json:"issuer"`
	CreatedAt  time.Time `json:"created_at"`
	Active     bool      `json:"active"`
}
