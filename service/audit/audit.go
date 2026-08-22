// Package audit 提供操作审计日志。
package audit

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"palworld-panel/internal/database"
)

// Record 审计记录。
type Record struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Detail    string    `json:"detail"`
	Result    string    `json:"result"`
	CreatedAt time.Time `json:"created_at"`
}

const tableName = "audit"

// Add 记录一条审计日志。
func Add(store *database.Store, source, action, target, detail, result string) error {
	if store == nil {
		return nil
	}
	r := Record{
		ID:        uuid.NewString(),
		Source:    source,
		Action:    action,
		Target:    target,
		Detail:    detail,
		Result:    result,
		CreatedAt: time.Now().UTC(),
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = store.DB().Exec(
		`INSERT INTO audit(actor, action, detail, ip, created_at) VALUES(?,?,?,?,?)`,
		r.Source, r.Action, string(data), "", r.CreatedAt)
	return err
}

// List 返回最近 limit 条审计记录。
func List(store *database.Store, limit int) ([]Record, error) {
	if store == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := store.DB().Query(
		`SELECT detail FROM audit ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []Record
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var r Record
		if err := json.Unmarshal([]byte(data), &r); err == nil {
			records = append(records, r)
		}
	}
	return records, rows.Err()
}
