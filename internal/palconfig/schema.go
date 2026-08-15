// Package palconfig 解析与生成 PalWorldSettings.ini。
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
	TypeRaw    FieldType = "raw"
)

func opt(value, label string) FieldOption {
	return FieldOption{Label: label, Value: value}
}

type FieldOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Field struct {
	Key             string        `json:"key"`
	Label           string        `json:"label"`
	Type            FieldType     `json:"type"`
	Default         string        `json:"default"`
	Description     string        `json:"description"`
	Options         []FieldOption `json:"options,omitempty"`
	Group           string        `json:"group"`
	RequiresRestart bool          `json:"requires_restart"`
	Min             *float64      `json:"min,omitempty"`
	Max             *float64      `json:"max,omitempty"`
	Step            *float64      `json:"step,omitempty"`
}

func fmin(v float64) *float64  { return &v }
func fmax(v float64) *float64  { return &v }
func fstep(v float64) *float64 { return &v }

// fRange 设置 min, max, step
func fRange(min, max, step float64) (*float64, *float64, *float64) {
	return &min, &max, &step
}

func rateField(key, label, def, desc, group string, restart bool, min, max float64) Field {
	f := Field{Key: key, Label: label, Type: TypeFloat, Default: def,
		Description: desc, Group: group, RequiresRestart: restart,
		Min: fmin(min), Step: fstep(0.1)}
	if max > 0 {
		f.Max = fmax(max)
	}
	return f
}

func intField(key, label, def, desc, group string, restart bool, min, max float64) Field {
	f := Field{Key: key, Label: label, Type: TypeInt, Default: def,
		Description: desc, Group: group, RequiresRestart: restart}
	if min >= 0 {
		f.Min = fmin(min)
	}
	if max > 0 {
		f.Max = fmax(max)
	}
	return f
}

func boolField(key, label, def, desc, group string, restart bool) Field {
	return Field{Key: key, Label: label, Type: TypeBool, Default: def,
		Description: desc, Group: group, RequiresRestart: restart}
}

func stringField(key, label, def, desc, group string) Field {
	return Field{Key: key, Label: label, Type: TypeString, Default: def,
		Description: desc, Group: group}
}

// Schema 全部配置项定义
func Schema() []Field {
	fields := []Field{
		// ---- 基础 ----
		stringField("ServerName", "服务器名称", "Default Palworld Server",
			"公开显示的服务器名称，会出现在服务器列表中", "服务器"),
		stringField("ServerDescription", "服务器描述", "",
			"社区服务器列表中显示的简介", "服务器"),
		{Key: "Difficulty", Label: "难度", Type: TypeEnum, Default: "None",
			Options: []FieldOption{opt("None", "自定义"), opt("Easy", "简单"), opt("Normal", "普通"), opt("Hard", "困难")},
			Description: "选择预设会自动填入官方倍率；修改任意倍率将自动切回自定义", Group: "服务器"},
		stringField("ServerPassword", "进服密码", "",
			"玩家进入服务器需要的密码，留空则无密码", "服务器"),
		stringField("AdminPassword", "管理员密码", "",
			"管理员权限、REST API 与 RCON 共同使用的密码；保存时自动同步到面板连接设置", "服务器"),
		intField("ServerPlayerMaxNum", "最大玩家数", "32",
			"同时在线最大人数（官方上限 32）", "服务器", false, 1, 32),
		intField("CoopPlayerMaxNum", "单人/合作人数上限", "4",
			"同一存档（公会）内的协作人数上限", "服务器", false, 1, 32),
		intField("GuildPlayerMaxNum", "单公会人数上限", "20",
			"单个公会最多可容纳的玩家数量", "服务器", false, 1, 32),
		intField("ChatPostLimitPerMinute", "每分钟聊天条数上限", "0",
			"单个玩家每分钟可发送的聊天条数，0 表示不限制", "服务器", false, 0, 0),
		{Key: "Region", Label: "地区", Type: TypeEnum, Default: "",
			Options: []FieldOption{opt("", "自动"), opt("Asia", "亚洲"), opt("Europe", "欧洲"), opt("NAmerica", "北美洲"), opt("SAmerica", "南美洲"), opt("Oceania", "大洋洲")},
			Description: "服务器地区，影响社区服务器列表分区", Group: "服务器"},

		// ---- 网络 ----
		intField("Port", "游戏端口", "8211",
			"玩家连接使用的 UDP 端口，需在防火墙/安全组放行", "服务器", true, 1, 65535),
		stringField("PublicIP", "公网 IP", "",
			"服务器对外的公网 IP，留空自动检测；NAT 环境建议手动填写纯 IP（不带 http://）", "服务器"),
		intField("PublicPort", "公网端口", "8211",
			"NAT/端口映射环境下对外暴露的端口", "服务器", false, 1, 65535),
		boolField("RESTAPIEnabled", "启用 REST API", "False",
			"面板读取在线玩家、封禁等数据依赖此项", "服务器", true),
		intField("RESTAPIPort", "REST API 端口", "8212",
			"REST API 监听的 TCP 端口", "服务器", true, 1, 65535),
		boolField("RCONEnabled", "启用 RCON", "False",
			"远程控制台，面板执行踢出、封禁、广播等命令需要开启", "服务器", true),
		intField("RCONPort", "RCON 端口", "25575",
			"RCON 服务监听的 TCP 端口", "服务器", true, 1, 65535),
		boolField("IsCommunityServer", "社区服务器", "False",
			"开启后服务器会显示在官方社区服务器列表中", "服务器", false),
		boolField("EnableConnectToSteamDedicatedServer", "启用 EOS 跨平台网络", "False",
			"允许 Xbox/PS5 等 Epic 账号玩家跨平台加入", "服务器", true),
		{Key: "CrossplayPlatforms", Label: "允许跨玩平台", Type: TypeRaw, Default: "(Steam,Xbox,PS5,Mac)",
			Description: "跨平台列表，格式如 (Steam,Xbox,PS5,Mac)，无需加引号", Group: "服务器", RequiresRestart: true},

		// ---- 游戏平衡（倍率字段官方未限定范围，仅设最小值和步进）----
		rateField("DayTimeSpeedRate", "白天流逝速度", "1.0", "游戏内白天时间流速倍率", "世界", false, 0.1, 5),
		rateField("NightTimeSpeedRate", "夜晚流逝速度", "1.0", "游戏内夜晚时间流速倍率", "世界", false, 0.1, 5),
		rateField("ExpRate", "经验获取倍率", "1.0", "玩家与帕鲁获得经验值的倍率", "世界", false, 0.1, 20),
		rateField("WorkSpeedRate", "工作速度倍率", "1.0", "帕鲁在据点工作的速度倍率", "世界", false, 0.1, 5),
		rateField("PalCaptureRate", "捕获概率倍率", "1.0", "捕获帕鲁的成功概率倍率", "世界", false, 0.5, 5),
		rateField("PalSpawnNumRate", "帕鲁出现数量倍率", "1.0", "野外帕鲁刷新数量倍率，数值越大服务器负载越高", "世界", false, 0.5, 5),
		{Key: "PalEggDefaultHatchingTime", Label: "巨大蛋孵化时间(小时)", Type: TypeFloat, Default: "72.0",
			Description: "巨大帕鲁蛋所需孵化时长（小时），其余蛋按比例缩放", Group: "世界", Min: fmin(0), Step: fstep(0.1)},
		rateField("PalDamageRateAttack", "帕鲁攻击伤害倍率", "1.0", "帕鲁造成的伤害倍率", "世界", false, 0.1, 5),
		rateField("PalDamageRateDefense", "帕鲁受到伤害倍率", "1.0", "帕鲁承受的伤害倍率，越小越耐打", "世界", false, 0.1, 5),
		rateField("PalAutoHPRegeneRate", "帕鲁自然回血倍率", "1.0", "帕鲁非睡眠状态下生命自然恢复倍率", "世界", false, 0.1, 5),
		rateField("PalAutoHpRegeneRateInSleep", "帕鲁睡眠回血倍率", "1.0", "帕鲁睡眠时生命恢复倍率（官方键名拼写为 Hp）", "世界", false, 0.1, 5),
		rateField("PalStaminaDecreaceRate", "帕鲁耐力消耗倍率", "1.0", "帕鲁耐力消耗速度倍率（官方键名拼写为 Decreace）", "世界", false, 0.1, 5),
		rateField("PalStomachDecreaceRate", "帕鲁饥饿消耗倍率", "1.0", "帕鲁饱食度下降速度倍率（官方键名拼写为 Decreace）", "世界", false, 0.1, 5),
		rateField("PlayerDamageRateAttack", "玩家攻击伤害倍率", "1.0", "玩家造成的伤害倍率", "世界", false, 0.1, 5),
		rateField("PlayerDamageRateDefense", "玩家受到伤害倍率", "1.0", "玩家承受的伤害倍率，越小越耐打", "世界", false, 0.1, 5),
		rateField("PlayerAutoHPRegeneRate", "玩家自然回血倍率", "1.0", "玩家非睡眠状态生命自然恢复倍率", "世界", false, 0.1, 5),
		rateField("PlayerAutoHpRegeneRateInSleep", "玩家睡眠回血倍率", "1.0", "玩家睡眠时生命恢复倍率（官方键名拼写为 Hp）", "世界", false, 0.1, 5),
		rateField("PlayerStaminaDecreaceRate", "玩家耐力消耗倍率", "1.0", "玩家耐力消耗速度倍率（官方键名拼写为 Decreace）", "世界", false, 0.1, 5),
		rateField("PlayerStomachDecreaceRate", "玩家饥饿消耗倍率", "1.0", "玩家饱食度下降速度倍率（官方键名拼写为 Decreace）", "世界", false, 0.1, 5),
		rateField("BuildObjectDamageRate", "建筑受伤倍率", "1.0", "建筑受到攻击时的伤害倍率", "世界", false, 0.5, 3),
		rateField("BuildObjectDeteriorationDamageRate", "建筑损坏速度倍率", "1.0", "建筑无人维护时自然劣化速度倍率", "世界", false, 0.1, 10),
		rateField("CollectionDropRate", "采集掉落倍率", "1.0", "砍伐、采矿等采集产出数量倍率", "世界", false, 0.5, 5),
		rateField("CollectionObjectHpRate", "采集物血量倍率", "1.0", "树木、矿点等可采集对象生命值倍率，越大越难采集", "世界", false, 0.5, 3),
		rateField("CollectionObjectRespawnSpeedRate", "采集物重生速度倍率", "1.0", "树木、矿石等重新刷新速度倍率", "世界", false, 0.5, 5),
		rateField("EnemyDropItemRate", "敌人掉落物倍率", "1.0", "击败敌人/帕鲁时掉落物品数量倍率", "世界", false, 0.5, 5),
		rateField("MonsterFarmActionSpeedRate", "牧场生产速度倍率", "1.0", "牧场帕鲁产出物品的速度倍率", "世界", false, 0.1, 10),
		rateField("ItemWeightRate", "物品重量倍率", "1.0", "物品重量倍率，越小可携带越多", "世界", false, 0.1, 5),
		rateField("ItemCorruptionMultiplier", "物品腐坏速度倍率", "1.0", "食物腐坏速度倍率", "世界", false, 0.1, 10),
		rateField("EquipmentDurabilityDamageRate", "装备耐久损耗倍率", "1.0", "武器、防具耐久度下降速度倍率", "世界", false, 0.1, 5),
		intField("SupplyDropSpan", "空投补给间隔(秒)", "300",
			"空投补给出现的间隔（秒）", "世界", false, 60, 0),
		intField("AutoSaveSpan", "自动存档间隔(秒)", "30",
			"服务器自动存档的时间间隔", "世界", false, 10, 0),

		// ---- 据点（官方明确范围）----
		intField("BaseCampMaxNum", "全服据点数上限", "128",
			"整个服务器最多可建立的据点数", "据点", false, 0, 0),
		intField("BaseCampMaxNumInGuild", "每公会据点数上限", "4",
			"每个公会最多可建立的据点数（官方上限 10）", "据点", false, 0, 10),
		intField("BaseCampWorkerMaxNum", "每据点工作帕鲁上限", "15",
			"每个据点最多可分配的工作帕鲁数（官方上限 50）", "据点", false, 0, 50),
		intField("MaxBuildingLimitNum", "每人建筑数量上限", "0",
			"每个玩家可建造建筑数量上限，0 表示不限制", "据点", false, 0, 0),

		// ---- 功能 ----
		boolField("bIsPvP", "PvP 模式", "False", "开启后玩家之间可以互相伤害", "多人与公会", false),
		boolField("bHardcore", "硬核模式", "False", "角色死亡后无法复活", "多人与公会", true),
		boolField("bPalLost", "死亡永久失去帕鲁", "False", "玩家死亡将永久失去其帕鲁", "多人与公会", false),
		boolField("bEnableFriendlyFire", "友军伤害", "False", "同公会/队友攻击也会造成伤害", "多人与公会", false),
		boolField("bCanPickupOtherGuildDeathPenaltyDrop", "可拾取他公会死亡掉落", "False",
			"允许拾取其他公会成员死亡惩罚掉落的物品", "多人与公会", false),
		boolField("bEnableInvaderEnemy", "启用入侵事件", "True", "允许敌人/自卫队随机入侵据点", "多人与公会", false),
		boolField("bEnableFastTravel", "启用快速旅行", "True", "允许使用传送点快速旅行", "多人与公会", false),
		boolField("bEnableFastTravelOnlyBaseCamp", "仅据点间快速旅行", "False", "限制快速旅行只能在据点（床）之间进行", "多人与公会", false),
		boolField("bExistPlayerAfterLogout", "离线角色留在世界", "False", "玩家登出后角色仍停留在游戏世界中", "多人与公会", false),
		boolField("bIsStartLocationSelectByMap", "允许选择出生地点", "True", "新玩家可在地图上选择初始出生区域", "多人与公会", false),
		boolField("bShowPlayerList", "ESC 菜单显示玩家列表", "True", "暂停/ESC 菜单中显示在线玩家列表", "多人与公会", false),
		boolField("bIsShowJoinLeftMessage", "显示进/退服消息", "True", "玩家加入或离开时在聊天框提示", "多人与公会", false),
		boolField("bEnableVoiceChat", "启用语音聊天", "False", "启用游戏内置语音聊天", "多人与公会", false),
		boolField("bBuildAreaLimit", "限制快速旅行点附近建造", "False", "只能在快速旅行点/据点附近建造", "多人与公会", false),
		boolField("bAllowGlobalPalboxImport", "允许全局帕鲁箱导入", "False", "允许从全局/其他存档导入帕鲁", "多人与公会", false),
		boolField("bAllowGlobalPalboxExport", "允许全局帕鲁箱导出", "False", "允许把帕鲁导出到全局", "多人与公会", false),
		boolField("bAutoResetGuildNoOnlinePlayers", "自动清理无人公会", "False", "公会长时间无人在线时自动解散", "多人与公会", false),
		{Key: "AutoResetGuildTimeNoOnlinePlayers", Label: "无人公会解散时间(小时)", Type: TypeFloat, Default: "72.0",
			Description: "公会所有成员离线多久后自动解散（小时）", Group: "多人与公会", Min: fmin(0), Step: fstep(1)},
		boolField("bIsRandomizerPalLevelRandom", "帕鲁等级完全随机", "False",
			"开启后野外帕鲁等级完全随机；关闭则在区域范围内随机", "多人与公会", false),

		// ---- 高级（1.0 正式版新增及较少使用的字段）----
		boolField("bIsUseBackupSaveData", "启用世界备份", "True", "启用时自动备份存档（增加磁盘负载）", "进阶设置", false),
		{Key: "LogFormatType", Label: "日志格式", Type: TypeEnum, Default: "Text",
			Options: []FieldOption{opt("Text", "文本"), opt("Json", "JSON")},
			Description: "服务器日志输出格式", Group: "进阶设置", RequiresRestart: true},
		{Key: "RandomizerType", Label: "随机化模式", Type: TypeEnum, Default: "None",
			Options: []FieldOption{opt("None", "无"), opt("Region", "按区域"), opt("All", "完全随机")},
			Description: "野外帕鲁等级随机化方式", Group: "进阶设置"},
		stringField("RandomizerSeed", "随机化种子", "", "随机化使用的种子，留空则随机", "进阶设置"),
		boolField("bUseAuth", "启用平台认证", "True", "启用 Steam/Epic 平台认证，关闭后允许离线玩家加入", "进阶设置", true),
		stringField("BanListURL", "封禁列表 URL", "https://api.palworldgame.com/api/banlist.txt",
			"官方封禁列表地址", "进阶设置"),
		boolField("bEnableBuildingPlayerUIdDisplay", "显示建筑创建者ID", "False",
			"在建筑上显示创建者的玩家 ID", "进阶设置", false),
		boolField("EnablePredatorBossPal", "启用掠食者头目帕鲁", "True",
			"允许野生掠食者头目出现", "进阶设置", false),
		boolField("bEnablePlayerToPlayerDamage", "玩家间伤害", "False", "允许玩家对其他玩家造成伤害（非PVP）", "进阶设置", false),
		boolField("bEnableDefenseOtherGuildPlayer", "他公会玩家防御", "False",
			"对其他公会玩家启用防御判定", "进阶设置", false),
		boolField("bEnableNonLoginPenalty", "离线惩罚", "True",
			"玩家长时间不登录时施加惩罚", "进阶设置", false),
		boolField("bCharacterRecreateInHardcore", "硬核模式可重建角色", "False",
			"硬核模式死亡后允许重建角色", "进阶设置", false),
		intField("DropItemMaxNum", "掉落物最大数量", "3000",
			"世界中同时存在的掉落物上限", "进阶设置", false, 0, 0),
		boolField("bActiveUNKO", "启用UNKO", "False", "未知功能（官方保留）", "进阶设置", false),
		intField("DropItemMaxNum_UNKO", "UNKO掉落物上限", "100", "UNKO 掉落物最大数量", "进阶设置", false, 0, 0),
		{Key: "DropItemAliveMaxHours", Label: "掉落物存活时间(小时)", Type: TypeFloat, Default: "1.0",
			Description: "掉落物在世界中存在的最长时间（小时）", Group: "进阶设置", Min: fmin(0.1), Step: fstep(0.1)},
		rateField("BuildObjectHpRate", "建筑血量倍率", "1.0", "建筑最大生命值倍率", "进阶设置", false, 0.5, 5),
		{Key: "ServerReplicatePawnCullDistance", Label: "帕鲁同步距离(cm)", Type: TypeInt, Default: "15000",
			Description: "玩家周围帕鲁的同步距离（厘米），官方范围 5000–15000", Group: "进阶设置",
			Min: fmin(5000), Max: fmax(15000), Step: fstep(100)},
		boolField("bAllowClientMod", "允许玩家使用 Mod", "True", "允许加载了 Mod 的玩家加入", "进阶设置", false),
		{Key: "VoiceChatMaxVolumeDistance", Label: "语音最大听到距离", Type: TypeInt, Default: "2000",
			Description: "语音聊天能听到的最大距离", Group: "进阶设置", Min: fmin(0)},
		{Key: "VoiceChatZeroVolumeDistance", Label: "语音静音距离", Type: TypeInt, Default: "5000",
			Description: "超过此距离完全听不到语音", Group: "进阶设置", Min: fmin(0)},
		{Key: "DenyTechnologyList", Label: "禁用科技列表", Type: TypeString, Default: "()",
			Description: "禁用的科技 ID 列表，格式如 (TechID1,TechID2)", Group: "进阶设置"},
		{Key: "GuildRejoinCooldownMinutes", Label: "公会重加冷却(分钟)", Type: TypeInt, Default: "0",
			Description: "退出公会后多久才能重新加入（分钟）", Group: "进阶设置", Min: fmin(0)},
		{Key: "BlockRespawnTime", Label: "死亡复活冷却(秒)", Type: TypeFloat, Default: "5.0",
			Description: "死亡后需要等待多久才能复活（秒）", Group: "进阶设置", Min: fmin(0)},
		{Key: "RespawnPenaltyDurationThreshold", Label: "复活惩罚阈值(秒)", Type: TypeFloat, Default: "0",
			Description: "应用复活时间倍率的连续死亡间隔阈值（秒）", Group: "进阶设置", Min: fmin(0)},
		{Key: "RespawnPenaltyTimeScale", Label: "复活惩罚时间倍率", Type: TypeFloat, Default: "2.0",
			Description: "短时间内连续死亡时复活等待时间的倍率", Group: "进阶设置", Min: fmin(0), Step: fstep(0.1)},
		boolField("bDisplayPvPItemNumOnWorldMap_BaseCamp", "地图显示据点PvP物品数", "False",
			"在世界地图上显示各据点的 PvP 专属物品数量", "进阶设置", false),
		boolField("bDisplayPvPItemNumOnWorldMap_Player", "地图显示玩家PvP物品数", "False",
			"在世界地图上显示玩家位置和 PvP 专属物品数量", "进阶设置", false),
		boolField("bAllowEnhanceStat_Health", "允许强化HP", "True", "允许玩家分配点数到生命值", "进阶设置", false),
		boolField("bAllowEnhanceStat_Attack", "允许强化攻击", "True", "允许玩家分配点数到攻击", "进阶设置", false),
		boolField("bAllowEnhanceStat_Stamina", "允许强化耐力", "True", "允许玩家分配点数到耐力", "进阶设置", false),
		boolField("bAllowEnhanceStat_Weight", "允许强化负重", "True", "允许玩家分配点数到负重", "进阶设置", false),
		boolField("bAllowEnhanceStat_WorkSpeed", "允许强化工作速度", "True", "允许玩家分配点数到工作速度", "进阶设置", false),
		boolField("bEnableAimAssistPad", "手柄瞄准辅助", "True", "启用手柄瞄准辅助", "进阶设置", false),
		boolField("bEnableAimAssistKeyboard", "键盘瞄准辅助", "False", "启用键盘鼠标瞄准辅助", "进阶设置", false),
	}

	// 添加死亡惩罚枚举
	deathPenalty := Field{Key: "DeathPenalty", Label: "死亡惩罚", Type: TypeEnum, Default: "All",
		Options: []FieldOption{opt("None", "无掉落"), opt("Item", "掉落物品（除装备）"), opt("ItemAndEquipment", "掉落全部物品"), opt("All", "掉落物品和帕鲁")},
		Description: "玩家死亡时掉落的内容", Group: "多人与公会"}
	fields = append(fields, deathPenalty)

	return fields
}

// GetField 根据 key 查找字段定义
func GetField(key string) *Field {
	for _, f := range Schema() {
		if f.Key == key {
			return &f
		}
	}
	return nil
}

// ParseFile 解析 PalWorldSettings.ini
func ParseFile(path string) (map[string]string, error) {
	result := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return result, nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inSettings := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") {
			inSettings = strings.Contains(line, "PalGameWorldSettings")
			continue
		}
		if inSettings && strings.HasPrefix(line, "OptionSettings=") {
			body := strings.TrimPrefix(line, "OptionSettings=")
			body = strings.Trim(body, "()")
			for _, pair := range splitOptions(body) {
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) == 2 {
					key := strings.TrimSpace(kv[0])
					val := strings.TrimSpace(kv[1])
					val = strings.Trim(val, `"`)
					result[key] = val
				}
			}
		}
	}
	return result, scanner.Err()
}

func splitOptions(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i, c := range s {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

// WriteFile 写入 PalWorldSettings.ini
func WriteFile(path string, settings map[string]string) error {
	schema := Schema()
	var pairs []string
	for _, f := range schema {
		v, ok := settings[f.Key]
		if !ok || v == "" {
			v = f.Default
		}
		if f.Type == TypeString || f.Type == TypeRaw {
			if f.Type == TypeString && !isQuoted(v) {
				v = `"` + v + `"`
			}
		}
		pairs = append(pairs, f.Key+"="+v)
	}
	content := fmt.Sprintf("[/Script/Pal.PalGameWorldSettings]\nOptionSettings=(%s)\n", strings.Join(pairs, ","))
	return os.WriteFile(path, []byte(content), 0644)
}

func isQuoted(s string) bool {
	return strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`)
}

// GetDefaults 返回默认配置
func GetDefaults() map[string]string {
	result := map[string]string{}
	for _, f := range Schema() {
		result[f.Key] = f.Default
	}
	return result
}

// FieldNames 用于反射
func FieldNames() []string {
	var names []string
	t := reflect.TypeOf(Field{})
	for i := 0; i < t.NumField(); i++ {
		names = append(names, t.Field(i).Name)
	}
	return names
}

// Parse 从 ini 内容解析配置
func Parse(content string) map[string]string {
	result := map[string]string{}
	inSettings := false
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			inSettings = strings.Contains(line, "PalGameWorldSettings")
			continue
		}
		if inSettings && strings.HasPrefix(line, "OptionSettings=") {
			body := strings.TrimPrefix(line, "OptionSettings=")
			body = strings.Trim(body, "()")
			for _, pair := range splitOptions(body) {
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) == 2 {
					key := strings.TrimSpace(kv[0])
					val := strings.TrimSpace(kv[1])
					val = strings.Trim(val, `"`)
					result[key] = val
				}
			}
		}
	}
	return result
}

// ValidateValue 校验字段值是否在范围内
func ValidateValue(f Field, v string) error {
	val := strings.Trim(v, `"`)
	switch f.Type {
	case TypeInt:
		n := 0
		fmt.Sscanf(val, "%d", &n)
		if f.Min != nil && float64(n) < *f.Min {
			return fmt.Errorf("%s 不能小于 %v", f.Label, *f.Min)
		}
		if f.Max != nil && *f.Max > 0 && float64(n) > *f.Max {
			return fmt.Errorf("%s 不能大于 %v", f.Label, *f.Max)
		}
	case TypeFloat:
		var n float64
		fmt.Sscanf(val, "%f", &n)
		if f.Min != nil && n < *f.Min {
			return fmt.Errorf("%s 不能小于 %v", f.Label, *f.Min)
		}
		if f.Max != nil && *f.Max > 0 && n > *f.Max {
			return fmt.Errorf("%s 不能大于 %v", f.Label, *f.Max)
		}
	}
	return nil
}

// SaveFile 保存配置到 ini 文件（WriteFile 的别名）
func SaveFile(path string, settings map[string]string) error {
	return WriteFile(path, settings)
}
