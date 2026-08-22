package service

import (
	"strings"

	"palworld-panel/internal/database"
)

// IsBanned 检查 UserId 或 IP 是否被封禁。
func IsBanned(store *database.Store, userId, ip string) (bool, string) {
	q := store.DB()
	uid := strings.TrimPrefix(userId, "steam_")
	identifiers := []string{
		"user|" + userId,
		"user|" + uid,
		"user|steam_" + uid,
	}
	if ip != "" {
		identifiers = append(identifiers, "ip|"+ip)
	}
	var (
		reason    string
		found     bool
	)
	for _, id := range identifiers {
		var r string
		err := q.QueryRow(
			`SELECT reason FROM banlist WHERE identifier=? AND active=1 LIMIT 1`, id,
		).Scan(&r)
		if err == nil {
			found = true
			reason = r
		}
	}
	return found, reason
}
