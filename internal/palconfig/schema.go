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
	TypeString FieldType = "string"
	TypeInt    FieldType = "int"
	TypeFloat  FieldType = "float"
	TypeBool   FieldType = "bool"
	TypeEnum   FieldType = "enum"
)

// Field 描述一个可配置项
type Field struct {
	Key             string    `json:"key"`
	Label           string    `json:"label"`
	Type            FieldType `json:"type"`
	Default         string    `json:"default"`
	Description     string    `json:"description"`
	Options         []string  `json:"options,omitempty"`
	Group           string    `json:"group"`
	RequiresRestart bool      `json:"requires_restart"`
}

// Schema 全部配置项定义（按官方 DefaultPalWorldSettings.ini 整理）
func Schema() []Field {
	return []Field{
		// ---- 基础 ----
		{Key: "ServerName", Label: "服务器名称", Type: TypeString, Default: "Default Palworld Server",
			Description: "公开显示的服务器名称，会出现在服务器列表中", Group: "基础"},
		{Key: "ServerDescription", Label: "服务器描述", Type: TypeString, Default: "",
			Description: "社区服务器列表中显示的简介", Group: "基础"},
		{Key: "Difficulty", Label: "难度", Type: TypeString, Default: "None",
			Description: "游戏难度预设，None 表示使用下方各项自定义倍率", Group: "基础"},
		{Key: "ServerPassword", Label: "进服密码", Type: TypeString, Default: "",
			Description: "玩家进入服务器需要的密码，留空则无密码", Group: "基础"},
		{Key: "AdminPassword", Label: "管理员密码", Type: TypeString, Default: "",
			Description: "管理员权限、REST API 与 RCON 共同使用的密码；保存时会自动同步到面板连接设置", Group: "基础"},
		{Key: "ServerPlayerMaxNum", Label: "最大玩家数", Type: TypeInt, Default: "32",
			Description: "服务器同时在线最大人数（官方上限 32）", Group: "基础"},
		{Key: "CoopPlayerMaxNum", Label: "单人/合作人数上限", Type: TypeInt, Default: "4",
			Description: "同一存档（公会）内的协作人数上限", Group: "基础"},
		{Key: "GuildPlayerMaxNum", Label: "单公会人数上限", Type: TypeInt, Default: "20",
			Description: "单个公会最多可容纳的玩家数量", Group: "基础"},
		{Key: "ChatPostLimitPerMinute", Label: "每分钟聊天条数上限", Type: TypeInt, Default: "0",
			Description: "单个玩家每分钟可发送的聊天条数，0 表示不限制", Group: "基础"},
		{Key: "Region", Label: "地区", Type: TypeString, Default: "",
			Description: "服务器地区标识，留空自动", Group: "基础"},

		// ---- 网络 ----
		{Key: "Port", Label: "游戏端口", Type: TypeInt, Default: "8211",
			Description: "玩家连接使用的 UDP 端口，需在防火墙/安全组放行", Group: "网络", RequiresRestart: true},
		{Key: "PublicIP", Label: "公网 IP", Type: TypeString, Default: "",
			Description: "服务器对外的公网 IP，留空则自动检测；NAT 环境建议手动填写", Group: "网络"},
		{Key: "PublicPort", Label: "公网端口", Type: TypeInt, Default: "8211",
			Description: "NAT/端口映射环境下对外暴露的端口，与游戏端口不一致时填写", Group: "网络"},
		{Key: "RESTAPIEnabled", Label: "启用 REST API", Type: TypeBool, Default: "False",
			Description: "面板读取在线玩家、封禁等数据依赖此项，使用面板请开启", Group: "网络", RequiresRestart: true},
		{Key: "RESTAPIPort", Label: "REST API 端口", Type: TypeInt, Default: "8212",
			Description: "REST API 监听的 TCP 端口，需与面板「系统设置」中的 REST 地址一致", Group: "网络", RequiresRestart: true},
		{Key: "RCONEnabled", Label: "启用 RCON", Type: TypeBool, Default: "False",
			Description: "远程控制台，面板执行踢出、封禁、广播等命令需要开启", Group: "网络", RequiresRestart: true},
		{Key: "RCONPort", Label: "RCON 端口", Type: TypeInt, Default: "25575",
			Description: "RCON 服务监听的 TCP 端口，需与面板「系统设置」中的 RCON 地址一致", Group: "网络", RequiresRestart: true},
		{Key: "IsCommunityServer", Label: "社区服务器", Type: TypeBool, Default: "False",
			Description: "开启后服务器会显示在官方社区服务器列表中", Group: "网络"},
		{Key: "EnableConnectToSteamDedicatedServer", Label: "启用 EOS 跨平台网络", Type: TypeBool, Default: "False",
			Description: "启用后允许 Xbox/PS5 等 Epic 账号玩家跨平台加入（EOS 网络）", Group: "网络", RequiresRestart: true},
		{Key: "CrossplayPlatforms", Label: "允许跨玩平台", Type: TypeString, Default: "(Steam,Xbox,PS5,Mac)",
			Description: "跨平台列表，格式：(Steam,Xbox,PS5,Mac)", Group: "网络", RequiresRestart: true},

		// ---- 游戏平衡 ----
		{Key: "DayTimeSpeedRate", Label: "白天流逝速度", Type: TypeFloat, Default: "1.0",
			Description: "游戏内白天时间流速倍率，越大白天越短", Group: "游戏平衡"},
		{Key: "NightTimeSpeedRate", Label: "夜晚流逝速度", Type: TypeFloat, Default: "1.0",
			Description: "游戏内夜晚时间流速倍率", Group: "游戏平衡"},
		{Key: "ExpRate", Label: "经验获取倍率", Type: TypeFloat, Default: "1.0",
			Description: "玩家与帕鲁获得经验值的倍率", Group: "游戏平衡"},
		{Key: "WorkSpeedRate", Label: "工作速度倍率", Type: TypeFloat, Default: "1.0",
			Description: "帕鲁在据点工作的速度倍率", Group: "游戏平衡"},
		{Key: "PalCaptureRate", Label: "捕获概率倍率", Type: TypeFloat, Default: "1.0",
			Description: "捕获帕鲁的成功概率倍率，越大越容易捕获", Group: "游戏平衡"},
		{Key: "PalSpawnNumRate", Label: "帕鲁出现数量倍率", Type: TypeFloat, Default: "1.0",
			Description: "野外帕鲁刷新数量倍率，数值越大服务器负载越高", Group: "游戏平衡"},
		{Key: "PalEggDefaultHatchingTime", Label: "巨大蛋孵化时间(小时)", Type: TypeFloat, Default: "72.0",
			Description: "巨大帕鲁蛋所需孵化时长（小时），其余蛋按比例缩放", Group: "游戏平衡"},
		{Key: "PalDamageRateAttack", Label: "帕鲁攻击伤害倍率", Type: TypeFloat, Default: "1.0",
			Description: "帕鲁造成的伤害倍率", Group: "游戏平衡"},
		{Key: "PalDamageRateDefense", Label: "帕鲁受到伤害倍率", Type: TypeFloat, Default: "1.0",
			Description: "帕鲁承受的伤害倍率，越小越耐打", Group: "游戏平衡"},
		{Key: "PalAutoHPRegeneRate", Label: "帕鲁自然回血倍率", Type: TypeFloat, Default: "1.0",
			Description: "帕鲁在非睡眠状态下的生命自然恢复倍率", Group: "游戏平衡"},
		{Key: "PalAutoHpRegeneRateInSleep", Label: "帕鲁睡眠回血倍率", Type: TypeFloat, Default: "1.0",
			Description: "帕鲁在睡眠时的生命恢复倍率（官方键名拼写为 Hp）", Group: "游戏平衡"},
		{Key: "PalStaminaDecreaceRate", Label: "帕鲁耐力消耗倍率", Type: TypeFloat, Default: "1.0",
			Description: "帕鲁耐力（体力）消耗速度倍率（官方键名拼写为 Decreace）", Group: "游戏平衡"},
		{Key: "PalStomachDecreaceRate", Label: "帕鲁饥饿消耗倍率", Type: TypeFloat, Default: "1.0",
			Description: "帕鲁饱食度下降速度倍率（官方键名拼写为 Decreace）", Group: "游戏平衡"},
		{Key: "PlayerDamageRateAttack", Label: "玩家攻击伤害倍率", Type: TypeFloat, Default: "1.0",
			Description: "玩家造成的伤害倍率", Group: "游戏平衡"},
		{Key: "PlayerDamageRateDefense", Label: "玩家受到伤害倍率", Type: TypeFloat, Default: "1.0",
			Description: "玩家承受的伤害倍率，越小越耐打", Group: "游戏平衡"},
		{Key: "PlayerAutoHPRegeneRate", Label: "玩家自然回血倍率", Type: TypeFloat, Default: "1.0",
			Description: "玩家在非睡眠状态下的生命自然恢复倍率", Group: "游戏平衡"},
		{Key: "PlayerAutoHpRegeneRateInSleep", Label: "玩家睡眠回血倍率", Type: TypeFloat, Default: "1.0",
			Description: "玩家在睡眠时的生命恢复倍率（官方键名拼写为 Hp）", Group: "游戏平衡"},
		{Key: "PlayerStaminaDecreaceRate", Label: "玩家耐力消耗倍率", Type: TypeFloat, Default: "1.0",
			Description: "玩家耐力消耗速度倍率（官方键名拼写为 Decreace）", Group: "游戏平衡"},
		{Key: "PlayerStomachDecreaceRate", Label: "玩家饥饿消耗倍率", Type: TypeFloat, Default: "1.0",
			Description: "玩家饱食度下降速度倍率（官方键名拼写为 Decreace）", Group: "游戏平衡"},
		{Key: "BuildObjectDamageRate", Label: "建筑受伤倍率", Type: TypeFloat, Default: "1.0",
			Description: "建筑受到攻击时的伤害倍率", Group: "游戏平衡"},
		{Key: "BuildObjectDeteriorationDamageRate", Label: "建筑损坏速度倍率", Type: TypeFloat, Default: "1.0",
			Description: "建筑无人维护时的自然劣化速度倍率", Group: "游戏平衡"},
		{Key: "CollectionDropRate", Label: "采集掉落倍率", Type: TypeFloat, Default: "1.0",
			Description: "砍伐、采矿等采集行为的产出数量倍率", Group: "游戏平衡"},
		{Key: "CollectionObjectHpRate", Label: "采集物血量倍率", Type: TypeFloat, Default: "1.0",
			Description: "树木、矿点等可采集对象的生命值倍率，越大越难采集", Group: "游戏平衡"},
		{Key: "CollectionObjectRespawnSpeedRate", Label: "采集物重生速度倍率", Type: TypeFloat, Default: "1.0",
			Description: "采集对象（树木、矿石等）重新刷新速度倍率", Group: "游戏平衡"},
		{Key: "EnemyDropItemRate", Label: "敌人掉落物倍率", Type: TypeFloat, Default: "1.0",
			Description: "击败敌人/帕鲁时掉落物品数量倍率", Group: "游戏平衡"},
		{Key: "MonsterFarmActionSpeedRate", Label: "牧场生产速度倍率", Type: TypeFloat, Default: "1.0",
			Description: "牧场帕鲁产出物品的速度倍率", Group: "游戏平衡"},
		{Key: "ItemWeightRate", Label: "物品重量倍率", Type: TypeFloat, Default: "1.0",
			Description: "物品重量倍率，越小可携带物品越多", Group: "游戏平衡"},
		{Key: "ItemCorruptionMultiplier", Label: "物品腐坏速度倍率", Type: TypeFloat, Default: "1.0",
			Description: "食物等可腐坏物品的腐坏速度倍率", Group: "游戏平衡"},
		{Key: "EquipmentDurabilityDamageRate", Label: "装备耐久损耗倍率", Type: TypeFloat, Default: "1.0",
			Description: "武器、防具等装备耐久度下降速度倍率", Group: "游戏平衡"},
		{Key: "SupplyDropSpan", Label: "空投补给间隔(秒)", Type: TypeInt, Default: "300",
			Description: "空投补给出现的间隔（秒）", Group: "游戏平衡"},
		{Key: "AutoSaveSpan", Label: "自动存档间隔(秒)", Type: TypeInt, Default: "30",
			Description: "服务器自动存档的时间间隔", Group: "游戏平衡"},

		// ---- 功能 ----
		{Key: "bIsPvP", Label: "PvP 模式", Type: TypeBool, Default: "False",
			Description: "开启后玩家之间可以互相伤害", Group: "功能"},
		{Key: "bHardcore", Label: "硬核模式", Type: TypeBool, Default: "False",
			Description: "硬核模式，角色死亡后无法复活", Group: "功能", RequiresRestart: true},
		{Key: "bPalLost", Label: "死亡永久失去帕鲁", Type: TypeBool, Default: "False",
			Description: "开启后玩家死亡将永久失去其帕鲁", Group: "功能"},
		{Key: "bEnableFriendlyFire", Label: "友军伤害", Type: TypeBool, Default: "False",
			Description: "开启后同公会/队友的攻击也会造成伤害", Group: "功能"},
		{Key: "bCanPickupOtherGuildDeathPenaltyDrop", Label: "可拾取他公会死亡掉落", Type: TypeBool, Default: "False",
			Description: "允许拾取其他公会成员死亡惩罚掉落的物品", Group: "功能"},
		{Key: "bEnableInvaderEnemy", Label: "启用入侵事件", Type: TypeBool, Default: "True",
			Description: "是否允许敌人/自卫队随机入侵据点", Group: "功能"},
		{Key: "bEnableFastTravel", Label: "启用快速旅行", Type: TypeBool, Default: "True",
			Description: "允许使用传送点进行快速旅行", Group: "功能"},
		{Key: "bEnableFastTravelOnlyBaseCamp", Label: "仅据点间快速旅行", Type: TypeBool, Default: "False",
			Description: "限制快速旅行只能在据点（床）之间进行", Group: "功能"},
		{Key: "bExistPlayerAfterLogout", Label: "离线角色留在世界", Type: TypeBool, Default: "False",
			Description: "玩家登出后其角色仍停留在游戏世界中", Group: "功能"},
		{Key: "bIsStartLocationSelectByMap", Label: "允许选择出生地点", Type: TypeBool, Default: "True",
			Description: "新玩家加入时可在地图上选择初始出生区域", Group: "功能"},
		{Key: "bShowPlayerList", Label: "ESC 菜单显示玩家列表", Type: TypeBool, Default: "True",
			Description: "在暂停/ESC 菜单中显示当前在线玩家列表", Group: "功能"},
		{Key: "bIsShowJoinLeftMessage", Label: "显示进/退服消息", Type: TypeBool, Default: "True",
			Description: "玩家加入或离开服务器时在聊天框提示", Group: "功能"},
		{Key: "bEnableVoiceChat", Label: "启用语音聊天", Type: TypeBool, Default: "False",
			Description: "启用游戏内置语音聊天", Group: "功能"},
		{Key: "bBuildAreaLimit", Label: "限制在快速旅行点附近建造", Type: TypeBool, Default: "False",
			Description: "开启后只能在快速旅行点/据点附近范围内建造", Group: "功能"},
		{Key: "bAllowGlobalPalboxImport", Label: "允许全局帕鲁箱导入", Type: TypeBool, Default: "False",
			Description: "允许从全局/其他存档导入帕鲁到帕鲁箱", Group: "功能"},
		{Key: "bAllowGlobalPalboxExport", Label: "允许全局帕鲁箱导出", Type: TypeBool, Default: "False",
			Description: "允许把帕鲁箱中的帕鲁导出到全局", Group: "功能"},
		{Key: "bAutoResetGuildNoOnlinePlayers", Label: "自动清理无人公会", Type: TypeBool, Default: "False",
			Description: "公会长时间无人在线时自动解散", Group: "功能"},
		{Key: "AutoResetGuildTimeNoOnlinePlayers", Label: "无人公会解散时间(小时)", Type: TypeFloat, Default: "72.0",
			Description: "公会所有成员离线多久后自动解散（小时）", Group: "功能"},

		// ---- 死亡/惩罚 ----
		{Key: "DeathPenalty", Label: "死亡惩罚", Type: TypeEnum, Default: "All",
			Options: []string{"None", "Item", "ItemAndEquipment", "All"},
			Description: "None:无掉落 Item:仅掉落物品 ItemAndEquipment:物品+装备 All:物品、装备、帕鲁全部掉落", Group: "死亡惩罚", RequiresRestart: true},
		{Key: "bEnableNonLoginPenalty", Label: "启用离线惩罚", Type: TypeBool, Default: "True",
			Description: "玩家长期不登录时施加惩罚（如公会据点相关）", Group: "死亡惩罚"},

		// ---- 性能 ----
		{Key: "BaseCampMaxNum", Label: "全服据点数上限", Type: TypeInt, Default: "128",
			Description: "整个服务器可存在的据点总数上限", Group: "性能", RequiresRestart: true},
		{Key: "BaseCampMaxNumInGuild", Label: "每公会据点数上限", Type: TypeInt, Default: "4",
			Description: "单个公会可拥有的据点数，最大 10，越大负载越高", Group: "性能", RequiresRestart: true},
		{Key: "BaseCampWorkerMaxNum", Label: "每据点工作帕鲁上限", Type: TypeInt, Default: "20",
			Description: "单个据点可同时工作的帕鲁数量上限", Group: "性能", RequiresRestart: true},
		{Key: "ServerReplicatePawnCullDistance", Label: "帕鲁同步距离(cm)", Type: TypeInt, Default: "10000",
			Description: "超出此距离（厘米）的帕鲁不向客户端同步，调小可提升性能但可见范围变近（5000–15000）", Group: "性能", RequiresRestart: true},
		{Key: "MaxBuildingLimitNum", Label: "每人建筑数量上限", Type: TypeInt, Default: "0",
			Description: "单个玩家可建造的建筑数量上限，0 表示无限制", Group: "性能"},
		{Key: "bIsUseBackupSaveData", Label: "备份存档", Type: TypeBool, Default: "True",
			Description: "自动备份存档数据，会额外占用磁盘空间", Group: "性能"},

		// ---- 其他 ----
		{Key: "BanListURL", Label: "封禁名单 URL", Type: TypeString,
			Default: "https://api.palworldgame.com/api/banlist.txt",
			Description: "全局封禁名单的下载地址", Group: "其他"},
		{Key: "LogFormatType", Label: "日志格式", Type: TypeEnum, Default: "Text",
			Options: []string{"Text", "Json"},
			Description: "服务器日志输出格式：Text 文本或 Json 结构化", Group: "其他"},
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
