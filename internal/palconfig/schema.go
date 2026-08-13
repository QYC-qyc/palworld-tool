// Package palconfig 解析与生成 PalWorldSettings.ini。
// 配置文件结构：
//
//	[/Script/Pal.PalGameWorldSettings]
//	OptionSettings=(Key1=Value1,Key2=Value2,...,KeyN=ValueN)
package palconfig

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"strings"
)

// FieldType 配置项类型
type FieldType string

const (
	TypeString  FieldType = "string"
	TypeInt     FieldType = "int"
	TypeFloat   FieldType = "float"
	TypeBool    FieldType = "bool"
	TypeEnum    FieldType = "enum"
)

// Field 描述一个可配置项
type Field struct {
	Key          string    `json:"key"`
	Label        string    `json:"label"`
	Type         FieldType `json:"type"`
	Default      string    `json:"default"`
	Description  string    `json:"description"`
	Options      []string  `json:"options,omitempty"`
	Group        string    `json:"group"`
	RequiresRestart bool   `json:"requires_restart"`
}

// Schema 全部配置项定义（按官方文档整理的常用项）
func Schema() []Field {
	return []Field{
		// ---- 基础 ----
		{Key: "ServerName", Label: "服务器名称", Type: TypeString, Default: "Default Palworld Server",
			Description: "公开显示的服务器名称", Group: "基础", RequiresRestart: false},
		{Key: "ServerDescription", Label: "服务器描述", Type: TypeString, Default: "",
			Description: "社区服务器列表中显示的描述", Group: "基础"},
		{Key: "ServerPassword", Label: "进服密码", Type: TypeString, Default: "",
			Description: "玩家进入服务器需要的密码，留空则无密码", Group: "基础"},
		{Key: "AdminPassword", Label: "管理员密码", Type: TypeString, Default: "",
			Description: "REST/RCON 与管理员权限使用的密码", Group: "基础"},
		{Key: "ServerPlayerMaxNum", Label: "最大玩家数", Type: TypeInt, Default: "32",
			Description: "同时在线最大人数", Group: "基础"},
		{Key: "GuildPlayerMaxNum", Label: "单公会人数上限", Type: TypeInt, Default: "4",
			Description: "每个公会最多人数", Group: "基础"},
		{Key: "ChatPostLimitPerMinute", Label: "每分钟聊天条数上限", Type: TypeInt, Default: "0",
			Description: "0 表示不限制", Group: "基础"},

		// ---- 网络 ----
		{Key: "RESTAPIEnabled", Label: "启用 REST API", Type: TypeBool, Default: "True",
			Description: "面板与外部工具读取数据需要", Group: "网络", RequiresRestart: true},
		{Key: "RESTAPIPort", Label: "REST API 端口", Type: TypeInt, Default: "8212",
			Description: "面板连接端口", Group: "网络", RequiresRestart: true},
		{Key: "RCONEnabled", Label: "启用 RCON", Type: TypeBool, Default: "False",
			Description: "远程控制台，踢封禁等需要", Group: "网络", RequiresRestart: true},
		{Key: "RCONPort", Label: "RCON 端口", Type: TypeInt, Default: "25575",
			Description: "RCON 服务端口", Group: "网络", RequiresRestart: true},
		{Key: "Port", Label: "游戏端口", Type: TypeInt, Default: "8211",
			Description: "玩家连接的 UDP 端口", Group: "网络", RequiresRestart: true},
		{Key: "PublicIP", Label: "公网IP", Type: TypeString, Default: "",
			Description: "服务器公网 IP，留空自动检测", Group: "网络"},
		{Key: "PublicPort", Label: "公网端口", Type: TypeInt, Default: "8211",
			Description: "NAT 环境下对外端口", Group: "网络"},
		{Key: "IsCommunityServer", Label: "社区服务器", Type: TypeBool, Default: "False",
			Description: "是否在社区服务器列表展示", Group: "网络"},
		{Key: "EnableConnectToSteamDedicatedServer", Label: "启用EOS网络", Type: TypeBool, Default: "False",
			Description: "跨平台 EOS 网络", Group: "网络"},

		// ---- 游戏平衡（官方文档字段）----
		{Key: "DayTimeSpeedRate", Label: "白天流逝速度", Type: TypeFloat, Default: "1.0",
			Description: "数值越大白天越快", Group: "游戏平衡"},
		{Key: "NightTimeSpeedRate", Label: "夜晚流逝速度", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "ExpRate", Label: "经验倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "PalCaptureRate", Label: "捕获概率倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "PalSpawnNumRate", Label: "帕鲁出现数量倍率", Type: TypeFloat, Default: "1.0",
			Description: "影响性能", Group: "游戏平衡"},
		{Key: "PalDamageRateAttack", Label: "帕鲁攻击伤害倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "PalDamageRateDefense", Label: "帕鲁受到伤害倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "PalAutoHPRegeneRate", Label: "帕鲁自然回血倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "PalAutoHpRegeneRateInSleep", Label: "帕鲁睡眠回血倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "PalStaminaDecreaceRate", Label: "帕鲁耐力消耗倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "PalStomachDecreaceRate", Label: "帕鲁饥饿消耗倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "PalEggDefaultHatchingTime", Label: "巨大蛋孵化时间(小时)", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "PlayerDamageRateAttack", Label: "玩家攻击伤害倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "PlayerDamageRateDefense", Label: "玩家受到伤害倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "PlayerAutoHPRegeneRate", Label: "玩家自然回血倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "PlayerAutoHpRegeneRateInSleep", Label: "玩家睡眠回血倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "PlayerStaminaDecreaceRate", Label: "玩家耐力消耗倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "PlayerStomachDecreaceRate", Label: "玩家饥饿消耗倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "BuildObjectDamageRate", Label: "建筑受伤倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "BuildObjectDeteriorationDamageRate", Label: "建筑损坏速度倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "CollectionDropRate", Label: "采集物掉落倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "CollectionObjectHpRate", Label: "采集物血量倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "CollectionObjectRespawnSpeedRate", Label: "采集物重生速度", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "EnemyDropItemRate", Label: "敌人掉落物倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "MonsterFarmActionSpeedRate", Label: "牧场生产速度倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "ItemWeightRate", Label: "物品重量倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "ItemCorruptionMultiplier", Label: "物品腐坏速度倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "EquipmentDurabilityDamageRate", Label: "装备耐久损耗倍率", Type: TypeFloat, Default: "1.0", Group: "游戏平衡"},
		{Key: "SupplyDropSpan", Label: "空投补给间隔(分钟)", Type: TypeInt, Default: "300", Group: "游戏平衡"},
		{Key: "AutoSaveSpan", Label: "自动存档间隔(秒)", Type: TypeInt, Default: "30", Group: "游戏平衡"},

		// ---- 功能（官方 Features）----
		{Key: "bIsPvP", Label: "PvP 模式", Type: TypeBool, Default: "False", Group: "功能"},
		{Key: "bHardcore", Label: "硬核模式", Type: TypeBool, Default: "False", Group: "功能", RequiresRestart: true},
		{Key: "bPalLost", Label: "死亡永久失去帕鲁", Type: TypeBool, Default: "False", Group: "功能"},
		{Key: "bEnableInvaderEnemy", Label: "启用入侵事件", Type: TypeBool, Default: "True", Group: "功能"},
		{Key: "bEnableFastTravel", Label: "启用快速旅行", Type: TypeBool, Default: "True", Group: "功能"},
		{Key: "bEnableFastTravelOnlyBaseCamp", Label: "仅据点间快速旅行", Type: TypeBool, Default: "False", Group: "功能"},
		{Key: "bExistPlayerAfterLogout", Label: "离线玩家角色留在世界", Type: TypeBool, Default: "False", Group: "功能"},
		{Key: "bShowPlayerList", Label: "ESC菜单显示玩家列表", Type: TypeBool, Default: "True", Group: "功能"},
		{Key: "bIsShowJoinLeftMessage", Label: "显示进/退服消息", Type: TypeBool, Default: "True", Group: "功能"},
		{Key: "bEnableVoiceChat", Label: "启用语音聊天", Type: TypeBool, Default: "False", Group: "功能"},
		{Key: "bBuildAreaLimit", Label: "限制在快速旅行点附近建造", Type: TypeBool, Default: "False", Group: "功能"},
		{Key: "bAllowGlobalPalboxImport", Label: "允许全局帕鲁箱导入", Type: TypeBool, Default: "False", Group: "功能"},
		{Key: "bAllowGlobalPalboxExport", Label: "允许全局帕鲁箱导出", Type: TypeBool, Default: "False", Group: "功能"},
		{Key: "bAutoResetGuildNoOnlinePlayers", Label: "自动清理无人公会", Type: TypeBool, Default: "False", Group: "功能"},
		{Key: "CrossplayPlatforms", Label: "允许跨玩平台", Type: TypeString, Default: "(Steam,Xbox,PS5,Mac)",
			Description: "格式：(Steam,Xbox,PS5,Mac)", Group: "功能", RequiresRestart: true},

		// ---- 死亡/惩罚 ----
		{Key: "DeathPenalty", Label: "死亡惩罚", Type: TypeEnum, Default: "All",
			Options: []string{"None", "Item", "ItemAndEquipment", "All"},
			Description: "None:无掉落 Item:仅物品 ItemAndEquipment:物品+装备 All:物品装备帕鲁全掉", Group: "死亡惩罚", RequiresRestart: true},
		{Key: "bEnableNonLoginPenalty", Label: "启用离线惩罚", Type: TypeBool, Default: "True", Group: "死亡惩罚"},

		// ---- 性能（官方 Performances）----
		{Key: "BaseCampMaxNum", Label: "全服据点数上限", Type: TypeInt, Default: "128", Group: "性能", RequiresRestart: true},
		{Key: "BaseCampMaxNumInGuild", Label: "每公会据点数上限", Type: TypeInt, Default: "4",
			Description: "最大 10，数值越大负载越高", Group: "性能", RequiresRestart: true},
		{Key: "BaseCampWorkerMaxNum", Label: "每据点帕鲁上限", Type: TypeInt, Default: "20",
			Description: "最大 50", Group: "性能", RequiresRestart: true},
		{Key: "ServerReplicatePawnCullDistance", Label: "帕鲁同步距离(cm)", Type: TypeInt, Default: "10000",
			Description: "5000-15000，越小性能越好但可见范围越近", Group: "性能", RequiresRestart: true},
		{Key: "bIsUseBackupSaveData", Label: "备份存档", Type: TypeBool, Default: "True",
			Description: "会增加磁盘负载", Group: "性能"},

		// ---- 其他 ----
		{Key: "BanListURL", Label: "封禁名单URL", Type: TypeString,
			Default: "https://api.palworldgame.com/api/banlist.txt", Group: "其他"},
		{Key: "LogFormatType", Label: "日志格式", Type: TypeEnum, Default: "Text",
			Options: []string{"Text", "Json"}, Group: "其他"},
	}
}

// Parse 从 ini 文件内容解析出 OptionSettings 键值对。
func Parse(content string) map[string]string {
	result := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "OptionSettings=") {
			continue
		}
		body := strings.TrimPrefix(line, "OptionSettings=")
		body = strings.TrimSuffix(strings.TrimPrefix(body, "("), ")")
		// 按逗号分割，但字符串值内可能含逗号——Palworld 配置中字符串用双引号
		for _, pair := range splitOptions(body) {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) != 2 {
				continue
			}
			k := strings.TrimSpace(kv[0])
			v := strings.TrimSpace(kv[1])
			result[k] = v
		}
	}
	return result
}

// splitOptions 按逗号分割选项，尊重引号内的逗号
func splitOptions(body string) []string {
	var parts []string
	var cur strings.Builder
	inQuote := false
	for _, r := range body {
		switch r {
		case '"':
			inQuote = !inQuote
			cur.WriteRune(r)
		case ',':
			if inQuote {
				cur.WriteRune(r)
			} else {
				parts = append(parts, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

// Serialize 将键值对写回 ini 文件内容。会保留段头格式。
func Serialize(settings map[string]string) string {
	var pairs []string
	for _, f := range Schema() {
		v, ok := settings[f.Key]
		if !ok {
			v = f.Default
		}
		pairs = append(pairs, fmt.Sprintf("%s=%s", f.Key, formatValue(f, v)))
	}
	// 保留未知的自定义键
	known := map[string]bool{}
	for _, f := range Schema() {
		known[f.Key] = true
	}
	for k, v := range settings {
		if !known[k] {
			pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
		}
	}

	return fmt.Sprintf("[/Script/Pal.PalGameWorldSettings]\nOptionSettings=(%s)\n", strings.Join(pairs, ","))
}

// formatValue 按类型格式化值（字符串加引号，布尔/数字直接写）
func formatValue(f Field, v string) string {
	switch f.Type {
	case TypeString:
		// 已经有引号就不再加
		if strings.HasPrefix(v, `"`) && strings.HasSuffix(v, `"`) {
			return v
		}
		return `"` + v + `"`
	case TypeBool:
		return capitalizeBool(v)
	default:
		return v
	}
}

func capitalizeBool(v string) string {
	switch strings.ToLower(v) {
	case "true", "1":
		return "True"
	default:
		return "False"
	}
}

// LoadFile 读取并解析 ini 文件
func LoadFile(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(b)), nil
}

// SaveFile 将配置写入 ini 文件
func SaveFile(path string, settings map[string]string) error {
	content := Serialize(settings)
	return os.WriteFile(path, []byte(content), 0644)
}

// ValidateValue 校验单个值
func ValidateValue(f Field, v string) error {
	switch f.Type {
	case TypeInt:
		var x int64
		if _, err := fmt.Sscan(v, &x); err != nil {
			return fmt.Errorf("%s 必须是整数", f.Label)
		}
	case TypeFloat:
		var x float64
		if _, err := fmt.Sscan(v, &x); err != nil {
			return fmt.Errorf("%s 必须是数字", f.Label)
		}
	case TypeEnum:
		for _, opt := range f.Options {
			if v == opt {
				return nil
			}
		}
		return fmt.Errorf("%s 必须是 %v 之一", f.Label, f.Options)
	case TypeBool:
		switch strings.ToLower(v) {
		case "true", "false", "1", "0":
		default:
			return fmt.Errorf("%s 必须是 True 或 False", f.Label)
		}
	}
	return nil
}

// 避免 reflect 未使用（保留供将来反射构造）
var _ = reflect.TypeOf
