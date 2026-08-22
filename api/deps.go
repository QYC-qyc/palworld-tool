package api

import (
	"palworld-panel/internal/config"
	"palworld-panel/internal/database"
	"palworld-panel/internal/gamesrv"
)

var (
	db           *database.Store
	cfg          *config.Config
	gameAPI      *gameServerAPI
	gameSettings = &gameSettingsAPI{}
)

// SetDeps 注入数据库与配置
func SetDeps(store *database.Store, c *config.Config) {
	db = store
	cfg = c
	gameAPI, _ = newGameServerAPI()
	if gameAPI != nil {
		gamesrv.Default = gameAPI.mgr
	}
	if gameAPI != nil && db != nil {
		gameAPI.mgr.SetConfig(loadGameServerConfig())
	}
}
