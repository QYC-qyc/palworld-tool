package anticheat

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"time"

	"go.etcd.io/bbolt"
	"paladmin/internal/database"
)

// SaveAlert 持久化一条告警，返回带 ID 的告警。写主表 + 二级索引
func SaveAlert(db *bbolt.DB, alert database.Alert) (database.Alert, error) {
	err := db.Update(func(tx *bbolt.Tx) error {
		mainB := tx.Bucket([]byte(database.BucketAlerts))
		seq, err := mainB.NextSequence()
		if err != nil {
			return err
		}
		alert.ID = seq
		if alert.CreatedAt.IsZero() {
			alert.CreatedAt = time.Now()
		}
		key := itob(seq)
		data, err := json.Marshal(alert)
		if err != nil {
			return err
		}
		if err := mainB.Put(key, data); err != nil {
			return err
		}
		if alert.PlayerUID != "" {
			ib, err := tx.CreateBucketIfNotExists([]byte(database.BucketAlertsPlayer))
			if err != nil {
				return err
			}
			idx := append([]byte(alert.PlayerUID+"|"), key...)
			_ = ib.Put(idx, key)
		}
		if alert.Status != "" {
			sb, err := tx.CreateBucketIfNotExists([]byte(database.BucketAlertsStatus))
			if err != nil {
				return err
			}
			idx := append([]byte(alert.Status+"|"), key...)
			_ = sb.Put(idx, key)
		}
		return nil
	})
	return alert, err
}

// ListAlerts 列出告警，支持 status/playerUID 过滤、limit、offset
func ListAlerts(db *bbolt.DB, status, playerUID string, limit, offset int) ([]database.Alert, int, error) {
	alerts := make([]database.Alert, 0)
	total := 0
	err := db.View(func(tx *bbolt.Tx) error {
		mainB := tx.Bucket([]byte(database.BucketAlerts))
		if mainB == nil {
			return nil
		}
		c := mainB.Cursor()
		// 倒序遍历（最新在前）
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			var a database.Alert
			if err := json.Unmarshal(v, &a); err != nil {
				continue
			}
			if status != "" && a.Status != status {
				continue
			}
			if playerUID != "" && a.PlayerUID != playerUID {
				continue
			}
			total++
			if offset > 0 {
				offset--
				continue
			}
			if limit > 0 && len(alerts) >= limit {
				continue
			}
			alerts = append(alerts, a)
		}
		return nil
	})
	return alerts, total, err
}

// GetAlert 获取单条告警
func GetAlert(db *bbolt.DB, id uint64) (database.Alert, error) {
	var a database.Alert
	err := db.View(func(tx *bbolt.Tx) error {
		v := tx.Bucket([]byte(database.BucketAlerts)).Get(itob(id))
		if v == nil {
			return errors.New("告警不存在")
		}
		return json.Unmarshal(v, &a)
	})
	return a, err
}

// UpdateAlertStatus 更新告警状态
func UpdateAlertStatus(db *bbolt.DB, id uint64, status, note string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		mainB := tx.Bucket([]byte(database.BucketAlerts))
		v := mainB.Get(itob(id))
		if v == nil {
			return errors.New("告警不存在")
		}
		var a database.Alert
		if err := json.Unmarshal(v, &a); err != nil {
			return err
		}
		// 删除旧 status 索引
		if a.Status != "" {
			sb := tx.Bucket([]byte(database.BucketAlertsStatus))
			if sb != nil {
				_ = sb.Delete(append([]byte(a.Status+"|"), itob(id)...))
			}
		}
		a.Status = status
		if note != "" {
			a.ActionTaken = note
		}
		data, err := json.Marshal(a)
		if err != nil {
			return err
		}
		if err := mainB.Put(itob(id), data); err != nil {
			return err
		}
		sb, err := tx.CreateBucketIfNotExists([]byte(database.BucketAlertsStatus))
		if err != nil {
			return err
		}
		return sb.Put(append([]byte(status+"|"), itob(id)...), itob(id))
	})
}

// AddAudit 写审计日志
func AddAudit(db *bbolt.DB, actor, action, target, detail, result string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketAudit))
		seq, err := b.NextSequence()
		if err != nil {
			return err
		}
		a := database.Audit{
			ID: seq, Actor: actor, Action: action, Target: target,
			Detail: detail, Result: result, CreatedAt: time.Now(),
		}
		data, err := json.Marshal(a)
		if err != nil {
			return err
		}
		return b.Put(itob(seq), data)
	})
}

// ListAudit 列出审计日志（倒序）
func ListAudit(db *bbolt.DB, limit int) ([]database.Audit, error) {
	if limit <= 0 {
		limit = 200
	}
	result := make([]database.Audit, 0, limit)
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketAudit))
		c := b.Cursor()
		for k, v := c.Last(); k != nil && len(result) < limit; k, v = c.Prev() {
			var a database.Audit
			if err := json.Unmarshal(v, &a); err == nil {
				result = append(result, a)
			}
		}
		return nil
	})
	return result, err
}

func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}
