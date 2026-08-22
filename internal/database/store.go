// Package database 提供持久化存储。
//
// 当前使用 SQLite（纯 Go 驱动 modernc.org/sqlite，无需 CGO）。
// 数据以 JSON blob 的形式按 bucket/key 存储；service 层通过 Store
// 接口读写，不直接依赖具体数据库。
package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// 表/集合名（沿用原 bbolt bucket 概念）
const (
	BucketPlayers   = "players"
	BucketGuilds    = "guilds"
	BucketWhitelist = "whitelist"
	BucketBackups   = "backups"
	BucketSettings  = "settings"
	BucketBanlist   = "banlist"
	BucketAudit     = "audit"
)

func allBuckets() []string {
	return []string{
		BucketPlayers, BucketGuilds, BucketBackups, BucketSettings,
		BucketWhitelist, BucketBanlist, BucketAudit,
	}
}

// Store 是数据库的高层接口，提供 JSON KV 存取。
// service 层依赖此接口而非 *sql.DB。
type Store struct {
	db   *sql.DB
	once sync.Once
}

var global *Store

// Get 返回全局 Store（由 InitDB 初始化）。
func Get() *Store { return global }

// InitDB 打开/创建 SQLite 数据库并建表。
func InitDB(path string) *Store {
	// 旧版使用 bbolt，其 .db 文件不是 SQLite 格式。若检测到旧文件，
	// 备份后重建，避免 SQLite 打开旧 bbolt 文件时 panic（file is not a database）。
	if path != "" && !isSQLiteFile(path) {
		if fi, err := os.Stat(path); err == nil && fi.Size() > 0 {
			bak := path + ".bbolt.bak"
			if renameErr := os.Rename(path, bak); renameErr == nil {
				fmt.Printf("检测到旧版 bbolt 数据库，已备份为 %s，将新建 SQLite 数据库\n", bak)
			} else {
				// 重命名失败则直接删除，确保能新建
				_ = os.Remove(path)
			}
		}
	}

	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		panic(err)
	}
	// SQLite 单写者，限制连接数避免 "database is locked"
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	s.migrate()
	global = s
	return s
}

// isSQLiteFile 判断文件是否为 SQLite 数据库（前 16 字节为 "SQLite format 3\000"）。
// 文件不存在时返回 true（视为新建，交给 SQLite 创建）。
func isSQLiteFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return true // 不存在 → 新建
	}
	defer f.Close()
	header := make([]byte, 16)
	n, _ := f.Read(header)
	if n == 0 {
		return true // 空文件 → 新建
	}
	return string(header[:n]) == "SQLite format 3\x00"
}

// migrate 建表。按实体建表，支持字段查询；JSON 字段保留原始数据。
func (s *Store) migrate() {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS players (
			player_uid  TEXT PRIMARY KEY,
			steam_id    TEXT NOT NULL DEFAULT '',
			nickname    TEXT NOT NULL DEFAULT '',
			data        TEXT NOT NULL,
			updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_players_steam ON players(steam_id);`,
		`CREATE INDEX IF NOT EXISTS idx_players_nickname ON players(nickname);`,

		`CREATE TABLE IF NOT EXISTS guilds (
			guild_id    TEXT PRIMARY KEY,
			name        TEXT NOT NULL DEFAULT '',
			data        TEXT NOT NULL,
			updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS whitelist (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT NOT NULL DEFAULT '',
			steam_id    TEXT NOT NULL,
			player_uid  TEXT NOT NULL DEFAULT '',
			created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(steam_id, player_uid)
		);`,

		`CREATE TABLE IF NOT EXISTS backups (
			backup_id   TEXT PRIMARY KEY,
			save_time   TIMESTAMP NOT NULL,
			path        TEXT NOT NULL,
			created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_backups_time ON backups(save_time);`,

		`CREATE TABLE IF NOT EXISTS settings (
			key         TEXT PRIMARY KEY,
			value       TEXT NOT NULL,
			updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS banlist (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			type        TEXT NOT NULL,
			identifier  TEXT NOT NULL,
			reason      TEXT NOT NULL DEFAULT '',
			issuer      TEXT NOT NULL DEFAULT '',
			created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			active      INTEGER NOT NULL DEFAULT 1
		);`,
		`CREATE INDEX IF NOT EXISTS idx_banlist_identifier ON banlist(identifier);`,
		`CREATE INDEX IF NOT EXISTS idx_banlist_active ON banlist(active);`,

		`CREATE TABLE IF NOT EXISTS audit (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			actor       TEXT NOT NULL DEFAULT '',
			action      TEXT NOT NULL,
			detail      TEXT NOT NULL DEFAULT '',
			ip          TEXT NOT NULL DEFAULT '',
			created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_audit_time ON audit(created_at);`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			panic(err)
		}
	}
}

// Close 关闭数据库。
func (s *Store) Close() error {
	if s != nil && s.db != nil {
		return s.db.Close()
	}
	return nil
}

// ---- 基础 KV 操作 ----

// Put 写入一个 JSON 值。
func (s *Store) Put(bucket, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO kv(bucket,key,value,updated_at) VALUES(?,?,?,?)
		 ON CONFLICT(bucket,key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`,
		bucket, key, string(data), time.Now().UTC(),
	)
	return err
}

// PutRaw 写入原始字符串。
func (s *Store) PutRaw(bucket, key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO kv(bucket,key,value,updated_at) VALUES(?,?,?,?)
		 ON CONFLICT(bucket,key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`,
		bucket, key, value, time.Now().UTC(),
	)
	return err
}

// GetRaw 读取原始字符串，不存在返回 ErrNotFound。
func (s *Store) GetRaw(bucket, key string) (string, error) {
	var val string
	err := s.db.QueryRow(`SELECT value FROM kv WHERE bucket=? AND key=?`, bucket, key).Scan(&val)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return val, err
}

// GetJSON 读取并反序列化到 dst。
func (s *Store) GetJSON(bucket, key string, dst interface{}) error {
	raw, err := s.GetRaw(bucket, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), dst)
}

// Delete 删除一个 key。
func (s *Store) Delete(bucket, key string) error {
	_, err := s.db.Exec(`DELETE FROM kv WHERE bucket=? AND key=?`, bucket, key)
	return err
}

// ListKeys 列出 bucket 下所有 key。
func (s *Store) ListKeys(bucket string) ([]string, error) {
	rows, err := s.db.Query(`SELECT key FROM kv WHERE bucket=? ORDER BY key`, bucket)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// ListAll 列出 bucket 下所有键值并反序列化到 out（out 必须是 map 指针或 slice 指针）。
func (s *Store) ListAll(bucket string, out interface{}) error {
	rows, err := s.db.Query(`SELECT value FROM kv WHERE bucket=? ORDER BY key`, bucket)
	if err != nil {
		return err
	}
	defer rows.Close()
	var raws []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return err
		}
		raws = append(raws, v)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	all := "[" + joinStrings(raws, ",") + "]"
	return json.Unmarshal([]byte(all), out)
}

// Count 返回 bucket 下条目数。
func (s *Store) Count(bucket string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM kv WHERE bucket=?`, bucket).Scan(&n)
	return n, err
}

// DB 暴露底层 *sql.DB，供需要自定义查询的 service 使用。
func (s *Store) DB() *sql.DB { return s.db }

// ErrNotFound 表示 key 不存在。
var ErrNotFound = errors.New("not found")

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	out := ss[0]
	for _, s := range ss[1:] {
		out += sep + s
	}
	return out
}
