// Package audit 提供通用操作审计日志
package audit

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

// Record 审计记录
type Record struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Detail    string    `json:"detail"`
	Result    string    `json:"result"`
	CreatedAt time.Time `json:"created_at"`
}

const bucketName = "audit"

// Add 记录一条审计日志
func Add(db *bbolt.DB, source, action, target, detail, result string) error {
	if db == nil {
		return nil
	}
	r := Record{
		ID:        uuid.NewString(),
		Source:    source,
		Action:    action,
		Target:    target,
		Detail:    detail,
		Result:    result,
		CreatedAt: time.Now(),
	}
	return db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}
		v, err := json.Marshal(r)
		if err != nil {
			return err
		}
		return b.Put([]byte(r.ID), v)
	})
}

// List 返回最近的审计记录
func List(db *bbolt.DB, limit int) ([]Record, error) {
	if db == nil {
		return nil, nil
	}
	var records []Record
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			var r Record
			if err := json.Unmarshal(v, &r); err == nil {
				records = append(records, r)
			}
			if limit > 0 && len(records) >= limit {
				break
			}
		}
		return nil
	})
	return records, err
}
