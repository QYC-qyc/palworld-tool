package config

import (
	"strings"

	"github.com/spf13/viper"
	"palworld-panel/internal/logger"
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
		Mode      string `mapstructure:"mode"`
		Service   string `mapstructure:"service"`
		Container string `mapstructure:"container"`
	} `mapstructure:"process"`

	Storage struct {
		Path                  string `mapstructure:"path"`
		SnapshotKeepPerPlayer int    `mapstructure:"snapshot_keep_per_player"`
		RetentionDays         int    `mapstructure:"retention_days"`
	} `mapstructure:"storage"`

	Log struct {
		Level           string `mapstructure:"level"`
		File            string `mapstructure:"file"`
		Chat            bool   `mapstructure:"chat"`
		Network         bool   `mapstructure:"network"`
		PlayerLogins    bool   `mapstructure:"player_logins"`
		PlayerDeaths    bool   `mapstructure:"player_deaths"`
		PlayerBuildings bool   `mapstructure:"player_buildings"`
		PlayerSummons   bool   `mapstructure:"player_summons"`
		PlayerCaptures  bool   `mapstructure:"player_captures"`
	} `mapstructure:"log"`
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
	viper.SetDefault("rest.username", "admin")
	viper.SetDefault("rest.timeout", 5)
	viper.SetDefault("save.sync_interval", 120)
	viper.SetDefault("save.backup_interval", 14400)
	viper.SetDefault("storage.path", "./pst.db")
	viper.SetDefault("storage.snapshot_keep_per_player", 200)
	viper.SetDefault("storage.retention_days", 90)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.file", "./logs/pst.log")
	viper.SetDefault("backup.keep_count", 48)
	viper.SetDefault("backup.keep_days", 14)
	viper.SetEnvPrefix("")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	viper.AutomaticEnv()

	if err := viper.Unmarshal(conf); err != nil {
		logger.Panicf("配置解析失败: %v", err)
	}
}

func webPort() int {
	return viper.GetInt("web.port")
}
