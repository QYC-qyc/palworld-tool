package database

import (
	"sync"
	"time"

	"go.etcd.io/bbolt"
	"paladmin/internal/logger"
)

var (
	db   *bbolt.DB
	once sync.Once
)

// Bucket 名称
const (
	BucketPlayers   = "players"
	BucketGuilds    = "guilds"
	BucketWhitelist = "whitelist"
	BucketBackups   = "backups"
	BucketSettings  = "settings"
	BucketBanlist   = "banlist"
	BucketAudit     = "audit"
)

func allBuckets() []string {
	return []string{
		BucketPlayers, BucketGuilds, BucketBackups, BucketSettings,
		BucketWhitelist, BucketBanlist, BucketAudit,
	}
}

// InitDB 打开/创建 bbolt 数据库并建立所有 bucket
func InitDB(path string) *bbolt.DB {
	db_, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 1 * time.Minute})
	if err != nil {
		logger.Panic(err)
	}
	for _, b := range allBuckets() {
		err = db_.Update(func(tx *bbolt.Tx) error {
			_, e := tx.CreateBucketIfNotExists([]byte(b))
			return e
		})
		if err != nil {
			logger.Panic(err)
		}
	}
	return db_
}

// GetDB 单例获取数据库
func GetDB(path string) *bbolt.DB {
	once.Do(func() {
		db = InitDB(path)
	})
	return db
}

// Close 关闭数据库
func Close() {
	if db != nil {
		_ = db.Close()
	}
}
