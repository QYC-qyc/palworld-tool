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
	"paladmin/internal/database"
	"paladmin/internal/logger"
	"paladmin/internal/system"
)

// RestoreBackup 回档：停服→备份当前存档→解压目标备份覆盖 Saved→启服。
// saveDir 为游戏 Saved 目录；backupPath 为备份 zip 文件名（位于工作目录 backups/ 下）。
func RestoreBackup(db *bbolt.DB, saveDir, backupPath string) error {
	if saveDir == "" {
		return fmt.Errorf("未配置 save.path，无法定位存档目录")
	}
	ctl := system.NewProcessCtl()
	logger.Warnf("开始回档，进程控制: %s", ctl.Name())

	// 1. 停服
	running, _ := ctl.IsRunning()
	if running {
		if err := ctl.Stop(); err != nil {
			return fmt.Errorf("停服失败: %w", err)
		}
		logger.Info("已发送停服指令，等待进程退出...")
		time.Sleep(5 * time.Second)
	}

	// 2. 安全网：回档前先把当前 Saved 备份一份
	rollbackName := fmt.Sprintf("rollback-%s", time.Now().Format("20060102-150405"))
	rollbackZip := filepath.Join(backupDir(), rollbackName+".zip")
	if _, err := os.Stat(saveDir); err == nil {
		if err := system.ZipDir(saveDir, rollbackZip); err != nil {
			logger.Errorf("回档前安全备份失败: %v（继续回档）", err)
		} else {
			_ = AddBackup(db, database.Backup{Path: rollbackName + ".zip"})
			logger.Infof("当前存档已安全备份为 %s", rollbackName)
		}
	}

	// 3. 解压目标备份覆盖 saveDir
	fullPath := backupPath
	if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Join(backupDir(), backupPath)
	}
	if err := unzipOverwrite(fullPath, saveDir); err != nil {
		// 回滚失败，尝试重启服务
		_ = ctl.Start()
		return fmt.Errorf("解压备份失败: %w", err)
	}
	logger.Infof("备份 %s 已解压到 %s", backupPath, saveDir)

	// 4. 启服
	if running {
		if err := ctl.Start(); err != nil {
			return fmt.Errorf("回档完成但启服失败，请手动启动: %w", err)
		}
		logger.Info("已发送启服指令")
	}
	return nil
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

func backupDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return "backups"
	}
	return filepath.Join(wd, "backups")
}
