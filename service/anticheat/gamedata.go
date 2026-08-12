package anticheat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"paladmin/internal/logger"
)

// GameData 合法数据集合，用于规则精确校验
type GameData struct {
	LegalPalIDs   map[string]bool
	BossPalIDs    map[string]bool
	LegalItemIDs  map[string]bool
	LegalPassives map[string]bool
	Limits        Limits
}

type PlayerLimits struct {
	MaxLevel       int `json:"max_level"`
	MaxExp         int64 `json:"max_exp"`
	MaxHP          int64 `json:"max_hp"`
	MaxStatusPoint int   `json:"max_status_point"`
}

type PalLimits struct {
	MaxLevel     int `json:"max_level"`
	MaxRank      int `json:"max_rank"`
	MaxHP        int64 `json:"max_hp"`
	MaxMelee     int   `json:"max_melee"`
	MaxRanged    int   `json:"max_ranged"`
	MaxDefense   int   `json:"max_defense"`
	MaxWorkspeed int   `json:"max_workspeed"`
	MaxTalent    int   `json:"max_talent"`
	MaxSoul      int   `json:"max_soul"`
	MaxPassives  int   `json:"max_passives"`
}

type ItemLimits struct {
	DefaultMaxStack  int            `json:"default_max_stack"`
	MaxStackOverrides map[string]int `json:"max_stack_overrides"`
}

type GuildLimits struct {
	MaxBaseCampLevel int `json:"max_base_camp_level"`
	MaxBases         int `json:"max_bases"`
}

type Limits struct {
	Player              PlayerLimits `json:"player"`
	Pal                 PalLimits    `json:"pal"`
	Item                ItemLimits   `json:"item"`
	Guild               GuildLimits  `json:"guild"`
	BossPrefixes        []string     `json:"boss_prefixes"`
	IllegalItemKeywords []string     `json:"illegal_item_keywords"`
}

var (
	gdOnce sync.Once
	gd     *GameData
)

// LoadGameData 从 dataDir 加载合法数据表（单例，可通过 ReloadGameData 重载）
func LoadGameData(dataDir string) *GameData {
	gdOnce.Do(func() {
		gd = &GameData{
			LegalPalIDs:   map[string]bool{},
			BossPalIDs:    map[string]bool{},
			LegalItemIDs:  map[string]bool{},
			LegalPassives: map[string]bool{},
		}
		gd.load(dataDir)
	})
	return gd
}

// ReloadGameData 强制重新加载（配置热更新用）
func ReloadGameData(dataDir string) *GameData {
	gd = &GameData{
		LegalPalIDs:   map[string]bool{},
		BossPalIDs:    map[string]bool{},
		LegalItemIDs:  map[string]bool{},
		LegalPassives: map[string]bool{},
	}
	gd.load(dataDir)
	gdOnce = sync.Once{}
	return gd
}

func (g *GameData) load(dataDir string) {
	readJSON := func(name string, v interface{}) {
		path := filepath.Join(dataDir, name)
		b, err := os.ReadFile(path)
		if err != nil {
			logger.Warnf("无法加载 %s: %v", name, err)
			return
		}
		if err := json.Unmarshal(b, v); err != nil {
			logger.Warnf("解析 %s 失败: %v", name, err)
		}
	}

	var palIDs struct {
		Legal []string `json:"legal"`
		Boss  []string `json:"boss"`
	}
	readJSON("pal_ids.json", &palIDs)
	for _, id := range palIDs.Legal {
		g.LegalPalIDs[id] = true
	}
	for _, id := range palIDs.Boss {
		g.BossPalIDs[id] = true
	}

	var itemIDs []string
	readJSON("item_ids.json", &itemIDs)
	for _, id := range itemIDs {
		g.LegalItemIDs[strings.ToLower(id)] = true
	}

	var passives []string
	readJSON("passive_ids.json", &passives)
	for _, id := range passives {
		g.LegalPassives[id] = true
	}

	readJSON("limits.json", &g.Limits)
	logger.Infof("游戏数据加载完成: %d 帕鲁, %d 物品, %d 词条",
		len(g.LegalPalIDs), len(g.LegalItemIDs), len(g.LegalPassives))
}

// IsLegalPalID 帕鲁类型是否合法（排除 Boss 变种）
func (g *GameData) IsLegalPalID(id string) bool {
	if id == "" || id == "Unknow" {
		return true
	}
	return g.LegalPalIDs[id]
}

// IsBossPalID 是否 Boss/塔主帕鲁
func (g *GameData) IsBossPalID(id string) bool {
	return g.BossPalIDs[id]
}

// IsLegalItem 物品是否合法（keyword 调试物品视为非法）
func (g *GameData) IsLegalItem(id string) bool {
	id = strings.ToLower(id)
	if id == "" || id == "none" {
		return true
	}
	for _, kw := range g.Limits.IllegalItemKeywords {
		if strings.Contains(strings.ToLower(id), strings.ToLower(kw)) {
			return false
		}
	}
	// 白名单为空时不阻断（数据不全），仅 keyword 过滤
	if len(g.LegalItemIDs) == 0 {
		return true
	}
	return g.LegalItemIDs[id]
}

// MaxStackFor 物品最大堆叠
func (g *GameData) MaxStackFor(itemID string) int {
	if v, ok := g.Limits.Item.MaxStackOverrides[itemID]; ok {
		return v
	}
	if g.Limits.Item.DefaultMaxStack > 0 {
		return g.Limits.Item.DefaultMaxStack
	}
	return 9999
}
