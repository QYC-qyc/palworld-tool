package service

import (
	"os"
	"path/filepath"
	"time"

	"palworld-panel/internal/database"
	"palworld-panel/internal/tool"
)

// PruneBackups 按保留策略清理旧备份：
//   - keepCount > 0 时，只保留最近 keepCount 个
//   - keepDays > 0 时，删除早于 keepDays 天的备份
//
// 返回删除的数量。同时删除磁盘上的 zip 文件和数据库记录。
func PruneBackups(store *database.Store, keepCount, keepDays int) (int, error) {
	backups, err := ListBackups(store)
	if err != nil {
		return 0, err
	}
	if len(backups) == 0 {
		return 0, nil
	}
	// ListBackups 已按时间倒序
	cutoff := time.Time{}
	if keepDays > 0 {
		cutoff = time.Now().Add(-time.Duration(keepDays) * 24 * time.Hour)
	}

	removed := 0
	for i, b := range backups {
		overCount := keepCount > 0 && i >= keepCount
		overAge := keepDays > 0 && b.SaveTime.Before(cutoff)
		if !overCount && !overAge {
			continue
		}
		// 删文件
		if b.Path != "" {
			_ = os.Remove(filepath.Join(tool.BackupDir(), b.Path))
		}
		_ = DeleteBackup(store, b.BackupId)
		removed++
	}
	return removed, nil
}
