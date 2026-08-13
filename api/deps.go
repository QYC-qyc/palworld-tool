package api

import (
	"go.etcd.io/bbolt"
	"paladmin/internal/config"
	"paladmin/service/anticheat"
)

var (
	db               *bbolt.DB
	engine           *anticheat.Engine
	cfg              *config.Config
	gameAPI          *gameServerAPI
	gameSettings = &gameSettingsAPI{}
)

// SetDeps 注入数据库、反作弊引擎与配置
func SetDeps(database *bbolt.DB, e *anticheat.Engine, c *config.Config) {
	db = database
	engine = e
	cfg = c
	gameAPI, _ = newGameServerAPI()
}
