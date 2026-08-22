package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"palworld-panel/internal/database"
)

// AddBackup 记录一次备份。
func AddBackup(store *database.Store, b database.Backup) error {
	if b.BackupId == "" {
		b.BackupId = uuid.NewString()
	}
	if b.SaveTime.IsZero() {
		b.SaveTime = time.Now()
	}
	data, err := json.Marshal(b)
	if err != nil {
		return err
	}
	_, err = store.DB().Exec(`
INSERT INTO backups(backup_id, save_time, path, created_at)
VALUES(?,?,?,?)
ON CONFLICT(backup_id) DO UPDATE SET save_time=excluded.save_time, path=excluded.path`,
		b.BackupId, b.SaveTime, string(data))
	// 注意：path 列存的是完整 JSON，保持与原实现一致
	_ = sql.ErrNoRows
	return err
}

// ListBackups 列出全部备份（按时间倒序）。
func ListBackups(store *database.Store) ([]database.Backup, error) {
	rows, err := store.DB().Query(`SELECT path FROM backups ORDER BY save_time DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []database.Backup
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var b database.Backup
		if err := json.Unmarshal([]byte(data), &b); err == nil {
			result = append(result, b)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].SaveTime.After(result[j].SaveTime)
	})
	return result, rows.Err()
}

// DeleteBackup 删除备份记录。
func DeleteBackup(store *database.Store, id string) error {
	_, err := store.DB().Exec(`DELETE FROM backups WHERE backup_id=?`, id)
	return err
}

// GetBackup 获取单个备份。
func GetBackup(store *database.Store, id string) (database.Backup, error) {
	var data string
	err := store.DB().QueryRow(`SELECT path FROM backups WHERE backup_id=?`, id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return database.Backup{}, ErrNoRecord
	}
	if err != nil {
		return database.Backup{}, err
	}
	var b database.Backup
	err = json.Unmarshal([]byte(data), &b)
	return b, err
}
