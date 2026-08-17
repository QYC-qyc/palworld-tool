package service

import (
	"encoding/json"
	"strings"

	"go.etcd.io/bbolt"
	"paladmin/internal/database"
)

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
