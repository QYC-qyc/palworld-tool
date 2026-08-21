package service

import (
	"encoding/json"

	"go.etcd.io/bbolt"
	"palworld-panel/internal/database"
)

// PutGuilds 批量写入公会
func PutGuilds(db *bbolt.DB, guilds []database.Guild) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketGuilds))
		for _, g := range guilds {
			v, err := json.Marshal(g)
			if err != nil {
				return err
			}
			if err := b.Put([]byte(g.AdminPlayerUid), v); err != nil {
				return err
			}
		}
		return nil
	})
}

// ListGuilds 列出全部公会
func ListGuilds(db *bbolt.DB) ([]database.Guild, error) {
	result := make([]database.Guild, 0)
	err := db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(database.BucketGuilds)).ForEach(func(k, v []byte) error {
			var g database.Guild
			if err := json.Unmarshal(v, &g); err == nil {
				result = append(result, g)
			}
			return nil
		})
	})
	return result, err
}

// GetGuild 获取单个公会
func GetGuild(db *bbolt.DB, adminUID string) (database.Guild, error) {
	var g database.Guild
	err := db.View(func(tx *bbolt.Tx) error {
		v := tx.Bucket([]byte(database.BucketGuilds)).Get([]byte(adminUID))
		if v == nil {
			return ErrNoRecord
		}
		return json.Unmarshal(v, &g)
	})
	return g, err
}
