package service

import (
	"go.etcd.io/bbolt"
	"paladmin/internal/database"
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

	SettingPalDefenderHost     = "paldefender.host"
	SettingPalDefenderPort     = "paldefender.port"
	SettingPalDefenderToken    = "paldefender.token"
	SettingPalDefenderBasePath = "paldefender.base_path"

	// 反作弊检测（外部监控与存档审计）
	SettingDetectEnabled      = "detect.enabled"        // 总开关
	SettingDetectBanOnDetect  = "detect.ban_on_detect"  // 检测到即封禁（否则仅踢出）
	SettingDetectKickOnDetect = "detect.kick_on_detect" // 检测到即踢出
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

		SettingDetectEnabled:      "true",
		SettingDetectBanOnDetect:  "false",
		SettingDetectKickOnDetect: "true",
	}
}

// InitSettings 首次启动时把 config.yaml 的值写入数据库（仅当该键不存在时）
func InitSettings(db *bbolt.DB, initial map[string]string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(database.BucketSettings))
		if err != nil {
			return err
		}
		for k, v := range initial {
			if b.Get([]byte(k)) == nil && v != "" {
				if err := b.Put([]byte(k), []byte(v)); err != nil {
					return err
				}
			} else if b.Get([]byte(k)) == nil {
				_ = b.Put([]byte(k), []byte(v))
			}
		}
		return nil
	})
}

// GetSetting 读取单个配置
func GetSetting(db *bbolt.DB, key string) string {
	var val string
	_ = db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketSettings))
		if b == nil {
			return nil
		}
		v := b.Get([]byte(key))
		if v != nil {
			val = string(v)
		}
		return nil
	})
	return val
}

// GetAllSettings 读取全部配置
func GetAllSettings(db *bbolt.DB) (map[string]string, error) {
	result := map[string]string{}
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketSettings))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			result[string(k)] = string(v)
			return nil
		})
	})
	return result, err
}

// SetSetting 更新单个配置
func SetSetting(db *bbolt.DB, key, value string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(database.BucketSettings))
		if err != nil {
			return err
		}
		return b.Put([]byte(key), []byte(value))
	})
}

// SetSettings 批量更新（传 JSON map）
func SetSettings(db *bbolt.DB, updates map[string]string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(database.BucketSettings))
		if err != nil {
			return err
		}
		for k, v := range updates {
			if err := b.Put([]byte(k), []byte(v)); err != nil {
				return err
			}
		}
		return nil
	})
}


