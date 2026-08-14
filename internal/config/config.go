package config

import (
	"strings"

	"github.com/spf13/viper"
	"paladmin/internal/logger"
)

// Config 全局配置，结构映射 config.yaml
type Config struct {
	Web struct {
		Password  string `mapstructure:"password"`
		Port      int    `mapstructure:"port"`
		Tls       bool   `mapstructure:"tls"`
		CertPath  string `mapstructure:"cert_path"`
		KeyPath   string `mapstructure:"key_path"`
		PublicUrl string `mapstructure:"public_url"`
	} `mapstructure:"web"`

	Task struct {
		SyncInterval        int    `mapstructure:"sync_interval"`
		PlayerLogging       bool   `mapstructure:"player_logging"`
		PlayerLoginMessage  string `mapstructure:"player_login_message"`
		PlayerLogoutMessage string `mapstructure:"player_logout_message"`
	} `mapstructure:"task"`

	Rcon struct {
		Address   string `mapstructure:"address"`
		Password  string `mapstructure:"password"`
		UseBase64 bool   `mapstructure:"use_base64"`
		Timeout   int    `mapstructure:"timeout"`
	} `mapstructure:"rcon"`

	Rest struct {
		Address  string `mapstructure:"address"`
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
		Timeout  int    `mapstructure:"timeout"`
	} `mapstructure:"rest"`

	Save struct {
		Path           string `mapstructure:"path"`
		DecodePath     string `mapstructure:"decode_path"`
		SyncInterval   int    `mapstructure:"sync_interval"`
		BackupInterval int    `mapstructure:"backup_interval"`
	} `mapstructure:"save"`

	Manage struct {
		KickNonWhitelist bool `mapstructure:"kick_non_whitelist"`
	} `mapstructure:"manage"`

	Backup struct {
		KeepCount int  `mapstructure:"keep_count"`
		KeepDays  int  `mapstructure:"keep_days"`
		OnBan     bool `mapstructure:"on_ban"`
	} `mapstructure:"backup"`

	Process struct {
		Mode      string `mapstructure:"mode"`       // noop / systemd / docker
		Service   string `mapstructure:"service"`    // systemd 服务名
		Container string `mapstructure:"container"`  // docker 容器名
	} `mapstructure:"process"`

	Storage struct {
		Path                  string `mapstructure:"path"`
		SnapshotKeepPerPlayer int    `mapstructure:"snapshot_keep_per_player"`
		RetentionDays         int    `mapstructure:"retention_days"`
	} `mapstructure:"storage"`

	Anticheat AnticheatConfig `mapstructure:"anticheat"`

	PalDefender PalDefenderConfig `mapstructure:"paldefender"`

	Log struct {
		Level          string `mapstructure:"level"`
		File           string `mapstructure:"file"`
		Chat           bool   `mapstructure:"chat"`
		Rcon           bool   `mapstructure:"rcon"`
		Network        bool   `mapstructure:"network"`
		PlayerLogins   bool   `mapstructure:"player_logins"`
		PlayerDeaths   bool   `mapstructure:"player_deaths"`
		PlayerBuildings bool  `mapstructure:"player_buildings"`
		PlayerSummons  bool   `mapstructure:"player_summons"`
		PlayerCaptures bool   `mapstructure:"player_captures"`
	} `mapstructure:"log"`
}

type PunishConfig struct {
	Warn            bool `mapstructure:"warn"`
	WarnWithReason  bool `mapstructure:"warn_with_reason"`
	Kick            bool `mapstructure:"kick"`
	Ban             bool `mapstructure:"ban"`
	IPBan           bool `mapstructure:"ipban"`
	Announce        bool `mapstructure:"announce"`
	BackupBeforeBan bool `mapstructure:"backup_before_ban"`
}

type AnticheatConfig struct {
	Enabled        bool                   `mapstructure:"enabled"`
	Mode           string                 `mapstructure:"mode"` // external / integrated
	ScanOnSaveSync bool                   `mapstructure:"scan_on_save_sync"`
	ScanLive       bool                   `mapstructure:"scan_live"`
	Cooldown       int                    `mapstructure:"cooldown"`
	Evidence       bool                   `mapstructure:"evidence"`
	EvidenceDir    string                 `mapstructure:"evidence_dir"`
	DataDir        string                 `mapstructure:"data_dir"`
	PalRulesDir    string                 `mapstructure:"pal_rules_dir"`
	Punish         PunishConfig           `mapstructure:"punish"`
	UseAdminWhitelist   bool              `mapstructure:"use_admin_whitelist"`
	AdminIPs            []string          `mapstructure:"admin_ips"`
	UseWhitelist        bool              `mapstructure:"use_whitelist"`
	WhitelistMessage    string            `mapstructure:"whitelist_message"`
	SteamIDProtection   bool              `mapstructure:"steamid_protection"`
	BannedNames         []string          `mapstructure:"banned_names"`
	BannedChatWords     []string          `mapstructure:"banned_chat_words"`
	BannedTechnologies  []string          `mapstructure:"banned_technologies"`
	IllegalPalStats     bool              `mapstructure:"illegal_pal_stats"`
	PalStatsMaxRank     int               `mapstructure:"pal_stats_max_rank"`
	BlockTowerBossCapture bool            `mapstructure:"block_tower_boss_capture"`
	ImportAction        string            `mapstructure:"import_action"` // block / clamp / remove_passive
	Rules               map[string]RuleOverride `mapstructure:"rules"`
}

type RuleOverride struct {
	Enabled  *bool                   `mapstructure:"enabled"`
	Severity string                  `mapstructure:"severity"`
	Actions  []string                `mapstructure:"actions"`
	Reason   string                  `mapstructure:"reason"`
	Params   map[string]interface{}  `mapstructure:"params"`
}

type PalDefenderConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	Address         string `mapstructure:"address"`
	Token           string `mapstructure:"token"`
	Timeout         int    `mapstructure:"timeout"`
	SyncBanlist     bool   `mapstructure:"sync_banlist"`
	SyncImportRules bool   `mapstructure:"sync_import_rules"`
}

var Conf Config

// Init 加载配置：指定文件 > ./config.yaml > 环境变量
func Init(cfgFile string, conf *Config) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		viper.SetConfigType("yaml")
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logger.Warn("未找到 config.yaml，尝试从环境变量读取")
		} else {
			logger.Panicf("配置文件读取失败: %s", err)
		}
	}

	// 默认值
	viper.SetDefault("web.port", 8080)
	viper.SetDefault("task.sync_interval", 60)
	viper.SetDefault("rcon.timeout", 5)
	viper.SetDefault("rcon.use_base64", false)
	viper.SetDefault("rest.username", "admin")
	viper.SetDefault("rest.timeout", 5)
	viper.SetDefault("save.sync_interval", 120)
	viper.SetDefault("save.backup_interval", 14400)
	viper.SetDefault("storage.path", "./pst.db")
	viper.SetDefault("storage.snapshot_keep_per_player", 200)
	viper.SetDefault("storage.retention_days", 90)
	viper.SetDefault("anticheat.enabled", true)
	viper.SetDefault("anticheat.mode", "external")
	viper.SetDefault("anticheat.scan_live", true)
	viper.SetDefault("anticheat.cooldown", 600)
	viper.SetDefault("anticheat.evidence", true)
	viper.SetDefault("anticheat.evidence_dir", "./evidence")
	viper.SetDefault("anticheat.data_dir", "./data/gamedata")
	viper.SetDefault("anticheat.pal_rules_dir", "./data/palrules")
	viper.SetDefault("anticheat.punish.warn", true)
	viper.SetDefault("anticheat.punish.warn_with_reason", true)
	viper.SetDefault("anticheat.punish.ban", true)
	viper.SetDefault("anticheat.punish.announce", true)
	viper.SetDefault("anticheat.punish.backup_before_ban", true)
	viper.SetDefault("anticheat.illegal_pal_stats", true)
	viper.SetDefault("anticheat.pal_stats_max_rank", -1)
	viper.SetDefault("anticheat.block_tower_boss_capture", true)
	viper.SetDefault("anticheat.import_action", "block")
	viper.SetDefault("anticheat.whitelist_message", "该服务器启用白名单，你不在名单中。")
	viper.SetDefault("paldefender.address", "http://127.0.0.1:17993")
	viper.SetDefault("paldefender.timeout", 5)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.file", "./logs/pst.log")
	viper.SetDefault("backup.keep_count", 48)
	viper.SetDefault("backup.keep_days", 14)

	viper.SetEnvPrefix("")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	viper.AutomaticEnv()

	if err := viper.Unmarshal(conf); err != nil {
		logger.Panicf("无法解析配置: %s", err)
	}
	Conf = *conf
}
