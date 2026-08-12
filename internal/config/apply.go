package config

import "github.com/spf13/viper"

// 已知布尔型设置键
var boolKeys = map[string]bool{
	"rcon.use_base64":    true,
	"anticheat.enabled":  true,
}

// ApplyToViper 把数据库里的运行时配置覆盖到 viper（仅内存，不写配置文件），
// 使各 tool 客户端立即使用最新值。
func ApplyToViper(settings map[string]string) {
	for k, v := range settings {
		if boolKeys[k] {
			viper.Set(k, v == "true" || v == "1" || v == "yes")
		} else {
			viper.Set(k, v)
		}
	}
}
