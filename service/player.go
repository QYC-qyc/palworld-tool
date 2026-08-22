package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"palworld-panel/internal/database"
)

// extractSteamID 从存档 platform_id（steam_xxx）提取纯 SteamID。
func extractSteamID(platformID string) string {
	if strings.HasPrefix(platformID, "steam_") {
		return strings.TrimPrefix(platformID, "steam_")
	}
	return ""
}

// PutPlayers 写入/更新存档解析出的玩家。保留已有的在线信息。
func PutPlayers(store *database.Store, players []database.Player) error {
	tx, err := store.DB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 查询已有数据用于合并在线字段
	getExisting, err := tx.Prepare(`SELECT data FROM players WHERE player_uid=?`)
	if err != nil {
		return err
	}
	defer getExisting.Close()
	upsert, err := tx.Prepare(`
INSERT INTO players(player_uid, steam_id, nickname, data, updated_at)
VALUES(?,?,?,?,CURRENT_TIMESTAMP)
ON CONFLICT(player_uid) DO UPDATE SET
	steam_id=excluded.steam_id,
	nickname=excluded.nickname,
	data=excluded.data,
	updated_at=CURRENT_TIMESTAMP`)
	if err != nil {
		return err
	}
	defer upsert.Close()

	for _, p := range players {
		var oldData string
		err := getExisting.QueryRow(p.PlayerUid).Scan(&oldData)
		if err == nil {
			var old database.Player
			if json.Unmarshal([]byte(oldData), &old) == nil {
				if old.SteamId != "" {
					p.SteamId = old.SteamId
				}
				p.Ip = old.Ip
				p.Ping = old.Ping
				p.LocationX = old.LocationX
				p.LocationY = old.LocationY
				p.LastOnline = old.LastOnline
			}
		}
		if p.SteamId == "" && p.PlatformID != "" {
			p.SteamId = extractSteamID(p.PlatformID)
		}
		if p.SaveLastOnline != "" {
			if t, err := time.Parse(time.RFC3339, p.SaveLastOnline); err == nil {
				p.LastOnline = t
			}
		}
		data, err := json.Marshal(p)
		if err != nil {
			return err
		}
		steamID := p.SteamId
		if strings.Contains(steamID, "000000") {
			steamID = ""
		}
		if _, err := upsert.Exec(p.PlayerUid, steamID, p.Nickname, string(data)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// PutPlayersOnline 更新在线玩家信息。
func PutPlayersOnline(store *database.Store, players []database.OnlinePlayer) error {
	tx, err := store.DB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, p := range players {
		if p.PlayerUid == "" {
			continue
		}
		var oldData string
		var player database.Player
		err := tx.QueryRow(`SELECT data FROM players WHERE player_uid=?`, p.PlayerUid).Scan(&oldData)
		if errors.Is(err, sql.ErrNoRows) {
			player = database.Player{
				TersePlayer: database.TersePlayer{
					PlayerUid: p.PlayerUid,
					OnlinePlayer: database.OnlinePlayer{
						SteamId:  p.SteamId,
						Nickname: p.Nickname,
					},
				},
			}
		} else if err != nil {
			return err
		} else if json.Unmarshal([]byte(oldData), &player) == nil {
			if player.SteamId == "" || strings.Contains(player.SteamId, "000000") {
				player.SteamId = p.SteamId
			}
		}
		player.Ip = p.Ip
		player.Ping = p.Ping
		player.LocationX = p.LocationX
		player.LocationY = p.LocationY
		player.Level = p.Level
		if p.Nickname != "" {
			player.Nickname = p.Nickname
		}
		player.LastOnline = time.Now()
		data, err := json.Marshal(player)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`
INSERT INTO players(player_uid, steam_id, nickname, data, updated_at)
VALUES(?,?,?,?,CURRENT_TIMESTAMP)
ON CONFLICT(player_uid) DO UPDATE SET data=excluded.data, updated_at=CURRENT_TIMESTAMP`,
			p.PlayerUid, player.SteamId, player.Nickname, string(data)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListPlayers 返回精简玩家列表（跳过 000000 占位 UID）。
func ListPlayers(store *database.Store) ([]database.TersePlayer, error) {
	rows, err := store.DB().Query(`SELECT data FROM players WHERE player_uid NOT LIKE '%000000%' ORDER BY nickname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []database.TersePlayer
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var p database.TersePlayer
		if err := json.Unmarshal([]byte(data), &p); err == nil {
			result = append(result, p)
		}
	}
	return result, rows.Err()
}

// GetPlayer 获取单个玩家详情。
func GetPlayer(store *database.Store, uid string) (database.Player, error) {
	var data string
	err := store.DB().QueryRow(`SELECT data FROM players WHERE player_uid=?`, uid).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return database.Player{}, ErrNoRecord
	}
	if err != nil {
		return database.Player{}, err
	}
	var p database.Player
	err = json.Unmarshal([]byte(data), &p)
	return p, err
}

// FindOnlinePlayerByUID 读取玩家在线信息。
func FindOnlinePlayerByUID(store *database.Store, uid string) (database.OnlinePlayer, error) {
	p, err := GetPlayer(store, uid)
	if err != nil {
		return database.OnlinePlayer{}, err
	}
	return p.OnlinePlayer, nil
}

// ---- 白名单 ----

// AddWhitelist 增加白名单玩家。
func AddWhitelist(store *database.Store, p database.PlayerW) error {
	id := whitelistID(p)
	_, err := store.DB().Exec(`
INSERT INTO whitelist(name, steam_id, player_uid) VALUES(?,?,?)
ON CONFLICT(steam_id, player_uid) DO UPDATE SET name=excluded.name`,
		p.Name, p.SteamID, p.PlayerUID)
	_ = id
	return err
}

// ListWhitelist 列出白名单。
func ListWhitelist(store *database.Store) ([]database.PlayerW, error) {
	rows, err := store.DB().Query(`SELECT name, steam_id, player_uid FROM whitelist ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []database.PlayerW
	for rows.Next() {
		var p database.PlayerW
		if err := rows.Scan(&p.Name, &p.SteamID, &p.PlayerUID); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// RemoveWhitelist 删除白名单玩家。
func RemoveWhitelist(store *database.Store, p database.PlayerW) error {
	res, err := store.DB().Exec(`DELETE FROM whitelist
		WHERE (player_uid!='' AND player_uid=?) OR (name!='' AND name=?) OR (steam_id!='' AND steam_id=?)`,
		p.PlayerUID, p.Name, p.SteamID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("未找到该玩家")
	}
	return nil
}

// PutWhitelist 批量覆盖白名单。
func PutWhitelist(store *database.Store, players []database.PlayerW) error {
	tx, err := store.DB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM whitelist`); err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO whitelist(name, steam_id, player_uid) VALUES(?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, p := range players {
		if _, err := stmt.Exec(p.Name, p.SteamID, p.PlayerUID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func whitelistID(p database.PlayerW) string {
	if p.PlayerUID != "" {
		return p.PlayerUID
	}
	return p.SteamID
}
