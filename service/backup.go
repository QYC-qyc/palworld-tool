package service

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/google/uuid"
	"go.etcd.io/bbolt"
	"paladmin/internal/database"
)

func AddBackup(db *bbolt.DB, b database.Backup) error {
	if b.BackupId == "" {
		b.BackupId = uuid.NewString()
	}
	if b.SaveTime.IsZero() {
		b.SaveTime = time.Now()
	}
	return db.Update(func(tx *bbolt.Tx) error {
		v, err := json.Marshal(b)
		if err != nil {
			return err
		}
		return tx.Bucket([]byte(database.BucketBackups)).Put([]byte(b.BackupId), v)
	})
}

func ListBackups(db *bbolt.DB) ([]database.Backup, error) {
	result := make([]database.Backup, 0)
	err := db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(database.BucketBackups)).ForEach(func(k, v []byte) error {
			var b database.Backup
			if err := json.Unmarshal(v, &b); err == nil {
				result = append(result, b)
			}
			return nil
		})
	})
	sort.Slice(result, func(i, j int) bool {
		return result[i].SaveTime.After(result[j].SaveTime)
	})
	return result, err
}

func DeleteBackup(db *bbolt.DB, id string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(database.BucketBackups)).Delete([]byte(id))
	})
}
