package service

import (
	"database/sql"
	"errors"

	"palworld-panel/internal/database"
)

// 可动态调整的运行时配置键
const (
	SettingWebPassword      = "web.password"
	SettingRestAddress      = "rest.address"
	SettingRestUsername     = "rest.username"
	SettingRestPassword     = "rest.password"
	SettingSavePath         = "save.path"
	SettingProcessMode      = "process.mode"
	SettingProcessService   = "process.service"
	SettingProcessContainer = "process.container"

	SettingPalDefenderHost          = "paldefender.host"
	SettingPalDefenderPort          = "paldefender.port"
	SettingPalDefenderToken         = "paldefender.token"
	SettingPalDefenderBasePath      = "paldefender.base_path"
	SettingPalDefenderAntiCheat     = "paldefender.anticheat_enabled"
	SettingPalDefenderCheatersKick  = "paldefender.cheaters_kick"
	SettingPalDefenderCheatersBan   = "paldefender.cheaters_ban"
	SettingPalDefenderCheatersIPBan = "paldefender.cheaters_ipban"
)

// DefaultSettings 返回需要在数据库中初始化的默认键（首次启动时从 viper 同步）
func DefaultSettings() map[string]string {
	return map[string]string{
		SettingWebPassword:      "",
		SettingRestAddress:      "",
		SettingRestUsername:     "admin",
		SettingRestPassword:     "",
		SettingSavePath:         "",
		SettingProcessMode:      "noop",
		SettingProcessService:   "palworld",
		SettingProcessContainer: "palworld",

		SettingPalDefenderHost:     "127.0.0.1",
		SettingPalDefenderPort:     "17993",
		SettingPalDefenderToken:    "",
		SettingPalDefenderBasePath: "/v1/pdapi",

		SettingPalDefenderAntiCheat:     "true",
		SettingPalDefenderCheatersKick:  "true",
		SettingPalDefenderCheatersBan:   "false",
		SettingPalDefenderCheatersIPBan: "false",
	}
}

func settingsTable() string { return database.BucketSettings }

// InitSettings 首次启动时写入默认值（已存在的键不覆盖）。
func InitSettings(store *database.Store, initial map[string]string) error {
	tx, err := store.DB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for k, v := range initial {
		_, err := tx.Exec(
			`INSERT OR IGNORE INTO settings(key,value) VALUES(?,?)`, k, v)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetSetting 读取单个配置。
func GetSetting(store *database.Store, key string) string {
	var val string
	err := store.DB().QueryRow(`SELECT value FROM settings WHERE key=?`, key).Scan(&val)
	if errors.Is(err, sql.ErrNoRows) {
		return ""
	}
	return val
}

// GetAllSettings 读取全部配置。
func GetAllSettings(store *database.Store) (map[string]string, error) {
	rows, err := store.DB().Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		result[k] = v
	}
	return result, rows.Err()
}

// SetSetting 更新单个配置。
func SetSetting(store *database.Store, key, value string) error {
	_, err := store.DB().Exec(
		`INSERT INTO settings(key,value) VALUES(?,?)
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=CURRENT_TIMESTAMP`,
		key, value)
	return err
}

// SetSettings 批量更新。
func SetSettings(store *database.Store, updates map[string]string) error {
	tx, err := store.DB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for k, v := range updates {
		if _, err := tx.Exec(
			`INSERT INTO settings(key,value) VALUES(?,?)
			 ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=CURRENT_TIMESTAMP`,
			k, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}
