package service

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"go.etcd.io/bbolt"
	"paladmin/internal/database"
)

// extractSteamID 从存档的 platform_id（steam_xxx / gdk_xxx）提取纯 SteamID
func extractSteamID(platformID string) string {
	if strings.HasPrefix(platformID, "steam_") {
		return strings.TrimPrefix(platformID, "steam_")
	}
	return ""
}

// PutPlayers 写入/更新存档解析出的玩家
func PutPlayers(db *bbolt.DB, players []database.Player) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketPlayers))
		for _, p := range players {
			existing := b.Get([]byte(p.PlayerUid))
			if existing != nil {
				var old database.Player
				if err := json.Unmarshal(existing, &old); err == nil {
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
			v, err := json.Marshal(p)
			if err != nil {
				return err
			}
			if err := b.Put([]byte(p.PlayerUid), v); err != nil {
				return err
			}
		}
		return nil
	})
}

// PutPlayersOnline 更新在线玩家信息
func PutPlayersOnline(db *bbolt.DB, players []database.OnlinePlayer) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketPlayers))
		for _, p := range players {
			var player database.Player
			existing := b.Get([]byte(p.PlayerUid))
			if existing == nil {
				player.PlayerUid = p.PlayerUid
				player.SteamId = p.SteamId
				player.Nickname = p.Nickname
			} else {
				if err := json.Unmarshal(existing, &player); err != nil {
					return err
				}
				if player.SteamId == "" || strings.Contains(player.SteamId, "000000") {
					player.SteamId = p.SteamId
				}
			}
			player.Ip = p.Ip
			player.Ping = p.Ping
			player.LocationX = p.LocationX
			player.LocationY = p.LocationY
			player.Level = p.Level
			player.LastOnline = time.Now()
			v, err := json.Marshal(player)
			if err != nil {
				return err
			}
			if err := b.Put([]byte(p.PlayerUid), v); err != nil {
				return err
			}
		}
		return nil
	})
}

// ListPlayers 返回精简玩家列表
func ListPlayers(db *bbolt.DB) ([]database.TersePlayer, error) {
	result := make([]database.TersePlayer, 0)
	err := db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(database.BucketPlayers)).ForEach(func(k, v []byte) error {
			if strings.Contains(string(k), "000000") {
				return nil
			}
			var p database.TersePlayer
			if err := json.Unmarshal(v, &p); err != nil {
				return err
			}
			result = append(result, p)
			return nil
		})
	})
	return result, err
}

// GetPlayer 获取单个玩家详情
func GetPlayer(db *bbolt.DB, uid string) (database.Player, error) {
	var p database.Player
	err := db.View(func(tx *bbolt.Tx) error {
		v := tx.Bucket([]byte(database.BucketPlayers)).Get([]byte(uid))
		if v == nil {
			return ErrNoRecord
		}
		return json.Unmarshal(v, &p)
	})
	return p, err
}

// FindOnlinePlayerByUID 从数据库读取在线玩家信息
func FindOnlinePlayerByUID(db *bbolt.DB, uid string) (database.OnlinePlayer, error) {
	p, err := GetPlayer(db, uid)
	if err != nil {
		return database.OnlinePlayer{}, err
	}
	return p.OnlinePlayer, nil
}

// AddWhitelist 增加白名单
func AddWhitelist(db *bbolt.DB, p database.PlayerW) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(database.BucketWhitelist))
		if err != nil {
			return err
		}
		key, err := findWhitelistKey(b, p)
		if err != nil {
			return err
		}
		data, err := json.Marshal(p)
		if err != nil {
			return err
		}
		if key != nil {
			return b.Put(key, data)
		}
		return b.Put([]byte(p.Name+"|"+p.SteamID+"|"+p.PlayerUID), data)
	})
}

// ListWhitelist 列出白名单
func ListWhitelist(db *bbolt.DB) ([]database.PlayerW, error) {
	result := make([]database.PlayerW, 0)
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketWhitelist))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var p database.PlayerW
			if err := json.Unmarshal(v, &p); err == nil {
				result = append(result, p)
			}
			return nil
		})
	})
	return result, err
}

// RemoveWhitelist 删除白名单
func RemoveWhitelist(db *bbolt.DB, p database.PlayerW) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketWhitelist))
		if b == nil {
			return errors.New("白名单为空")
		}
		key, err := findWhitelistKey(b, p)
		if err != nil || key == nil {
			return errors.New("未找到该玩家")
		}
		return b.Delete(key)
	})
}

// PutWhitelist 批量覆盖白名单
func PutWhitelist(db *bbolt.DB, players []database.PlayerW) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(database.BucketWhitelist))
		if err != nil {
			return err
		}
		if err := b.ForEach(func(k, _ []byte) error { return b.Delete(k) }); err != nil {
			return err
		}
		for _, p := range players {
			data, err := json.Marshal(p)
			if err != nil {
				return err
			}
			id := p.PlayerUID
			if id == "" {
				id = p.SteamID
			}
			if id == "" {
				continue
			}
			if err := b.Put([]byte(id), data); err != nil {
				return err
			}
		}
		return nil
	})
}

func findWhitelistKey(b *bbolt.Bucket, p database.PlayerW) ([]byte, error) {
	var found []byte
	err := b.ForEach(func(k, v []byte) error {
		var ex database.PlayerW
		if err := json.Unmarshal(v, &ex); err != nil {
			return err
		}
		if (p.PlayerUID != "" && ex.PlayerUID == p.PlayerUID) ||
			(p.Name != "" && ex.Name == p.Name) ||
			(p.SteamID != "" && ex.SteamID == p.SteamID) {
			found = append([]byte(nil), k...)
			return errors.New("found")
		}
		return nil
	})
	if err != nil && err.Error() == "found" {
		return found, nil
	}
	return nil, err
}
