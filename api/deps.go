package api

import (
	"go.etcd.io/bbolt"
	"paladmin/internal/config"
	"paladmin/service"
)

var (
	db           *bbolt.DB
	cfg          *config.Config
	gameAPI      *gameServerAPI
	gameSettings = &gameSettingsAPI{}
)

// SetDeps 注入数据库与配置
func SetDeps(database *bbolt.DB, c *config.Config) {
	db = database
	cfg = c
	gameAPI, _ = newGameServerAPI()
	if gameAPI != nil && db != nil {
		gameAPI.mgr.SetConfig(loadGameServerConfig())
		gameAPI.mgr.SetSettingGetter(func(k string) string { return service.GetSetting(db, k) })
	}
}
