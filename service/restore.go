package service

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.etcd.io/bbolt"
	"palworld-panel/internal/database"
	"palworld-panel/internal/gamesrv"
	"palworld-panel/internal/logger"
	"palworld-panel/internal/system"
	"palworld-panel/internal/tool"
)

// RestoreBackup 回档：停服→备份当前存档→解压目标备份→推送回游戏目录→启服。
// backupPath 为备份 zip 文件名（位于持久化备份目录下）或绝对路径。
func RestoreBackup(db *bbolt.DB, saveDir, backupPath string) error {
	fullPath := backupPath
	if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Join(tool.BackupDir(), backupPath)
	}
	if _, err := os.Stat(fullPath); err != nil {
		return fmt.Errorf("备份文件不存在: %w", err)
	}

	// 1. 停服（优先用 gamesrv.Manager，回退到进程控制配置）
	wasRunning, stopFn, startFn := gameControl()
	if wasRunning {
		logger.Info("回档：停止游戏服...")
		if err := stopFn(); err != nil {
			return fmt.Errorf("停服失败: %w", err)
		}
		time.Sleep(5 * time.Second)
	}

	// 2. 安全网：回档前先把当前存档备份一份（直接用面板备份逻辑）
	rollbackName := ""
	if gamesrv.Default != nil {
		if name, err := tool.Backup(); err == nil {
			rollbackName = name
			_ = AddBackup(db, database.Backup{Path: name})
			logger.Infof("回档前当前存档已安全备份为 %s", name)
		} else {
			logger.Errorf("回档前安全备份失败: %v（继续回档）", err)
		}
	}

	// 3. 解压目标备份到临时目录，再推送回游戏目录
	tmpRoot, err := os.MkdirTemp("", "paladm-restore-")
	if err != nil {
		_ = startFn()
		return err
	}
	defer os.RemoveAll(tmpRoot)

	// zip 内顶层为 Saved/，解压到 tmpRoot/Pal/ 下得到 tmpRoot/Pal/Saved
	palDir := filepath.Join(tmpRoot, "Pal")
	if err := unzipOverwrite(fullPath, palDir); err != nil {
		if wasRunning {
			_ = startFn()
		}
		return fmt.Errorf("解压备份失败: %w", err)
	}

	if gamesrv.Default != nil {
		if err := gamesrv.Default.PushLocalToSaved(tmpRoot); err != nil {
			if wasRunning {
				_ = startFn()
			}
			return fmt.Errorf("写入游戏目录失败: %w", err)
		}
	} else {
		// 本地模式：直接覆盖 saveDir
		_ = os.RemoveAll(saveDir)
		if err := copyTree(filepath.Join(palDir, "Saved"), saveDir); err != nil {
			if wasRunning {
				_ = startFn()
			}
			return err
		}
	}
	logger.Infof("备份 %s 已恢复到游戏存档", backupPath)

	// 4. 启服
	if wasRunning {
		logger.Info("回档：启动游戏服...")
		if err := startFn(); err != nil {
			return fmt.Errorf("回档完成但启服失败，请手动启动: %w", err)
		}
	}
	_ = rollbackName
	return nil
}

// gameControl 返回 (是否在运行, 停服函数, 启服函数)。
func gameControl() (bool, func() error, func() error) {
	if gamesrv.Default != nil {
		st, _ := gamesrv.Default.GetStatus()
		running := st != nil && st.Running
		return running, gamesrv.Default.Stop, gamesrv.Default.Start
	}
	ctl := system.NewProcessCtl()
	running, _ := ctl.IsRunning()
	return running, ctl.Stop, ctl.Start
}

// unzipOverwrite 解压 zip 到目标目录，覆盖同名文件
func unzipOverwrite(src, dst string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	for _, f := range r.File {
		target := filepath.Join(dst, f.Name)
		// 防 zip slip
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dst)+string(os.PathSeparator)) &&
			filepath.Clean(target) != filepath.Clean(dst) {
			return fmt.Errorf("非法 zip 路径: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := copyZipEntry(f, target); err != nil {
			return err
		}
	}
	return nil
}

func copyZipEntry(f *zip.File, dst string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	return err
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// BackupDir 返回备份目录（持久化）
func BackupDir() string { return tool.BackupDir() }
