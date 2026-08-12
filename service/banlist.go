package service

import (
	"encoding/json"
	"strings"
	"time"

	"go.etcd.io/bbolt"
	"paladmin/internal/database"
)

// AddBan 新增封禁记录（key: type|identifier）
func AddBan(db *bbolt.DB, record database.BanRecord) error {
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}
	record.Active = true
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketBanlist))
		key := string(record.Type) + "|" + record.Identifier
		v, err := json.Marshal(record)
		if err != nil {
			return err
		}
		return b.Put([]byte(key), v)
	})
}

// RemoveBan 解封
func RemoveBan(db *bbolt.DB, banType database.BanType, identifier string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(database.BucketBanlist)).Delete([]byte(string(banType) + "|" + identifier))
	})
}

// ListBans 列出封禁，可按 active 过滤
func ListBans(db *bbolt.DB, activeOnly bool) ([]database.BanRecord, error) {
	result := make([]database.BanRecord, 0)
	err := db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(database.BucketBanlist)).ForEach(func(k, v []byte) error {
			var r database.BanRecord
			if err := json.Unmarshal(v, &r); err != nil {
				return nil
			}
			if activeOnly && !r.Active {
				return nil
			}
			result = append(result, r)
			return nil
		})
	})
	return result, err
}

// IsBanned 检查 UserId 或 IP 是否被封禁
func IsBanned(db *bbolt.DB, userId, ip string) (bool, string) {
	found := false
	reason := ""
	_ = db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketBanlist))
		check := func(key string) bool {
			v := b.Get([]byte(key))
			if v == nil {
				return false
			}
			var r database.BanRecord
			if json.Unmarshal(v, &r) == nil && r.Active {
				reason = r.Reason
				return true
			}
			return false
		}
		uid := strings.TrimPrefix(userId, "steam_")
		if check("user|"+userId) || check("user|"+uid) || check("user|steam_"+uid) || (ip != "" && check("ip|"+ip)) {
			found = true
		}
		return nil
	})
	return found, reason
}
