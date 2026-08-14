package config

import (
	"github.com/spf13/viper"
)

// ApplyToViper 把数据库中的键值设置回 viper，使运行时读取以面板设置为准
func ApplyToViper(settings map[string]string) {
	for k, v := range settings {
		viper.Set(k, v)
	}
}
