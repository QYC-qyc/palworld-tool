package service

import (
	"database/sql"
	"encoding/json"
	"errors"

	"palworld-panel/internal/database"
)

// ErrNoRecord 记录不存在。
var ErrNoRecord = errors.New("record not found")

// PutGuilds 批量写入公会。
func PutGuilds(store *database.Store, guilds []database.Guild) error {
	tx, err := store.DB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`
INSERT INTO guilds(guild_id, name, data, updated_at)
VALUES(?,?,?,CURRENT_TIMESTAMP)
ON CONFLICT(guild_id) DO UPDATE SET name=excluded.name, data=excluded.data, updated_at=CURRENT_TIMESTAMP`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, g := range guilds {
		data, err := json.Marshal(g)
		if err != nil {
			return err
		}
		if _, err := stmt.Exec(g.AdminPlayerUid, g.Name, string(data)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListGuilds 列出全部公会。
func ListGuilds(store *database.Store) ([]database.Guild, error) {
	rows, err := store.DB().Query(`SELECT data FROM guilds ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []database.Guild
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var g database.Guild
		if err := json.Unmarshal([]byte(data), &g); err == nil {
			result = append(result, g)
		}
	}
	return result, rows.Err()
}

// GetGuild 获取单个公会。
func GetGuild(store *database.Store, adminUID string) (database.Guild, error) {
	var data string
	err := store.DB().QueryRow(`SELECT data FROM guilds WHERE guild_id=?`, adminUID).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return database.Guild{}, ErrNoRecord
	}
	if err != nil {
		return database.Guild{}, err
	}
	var g database.Guild
	err = json.Unmarshal([]byte(data), &g)
	return g, err
}
