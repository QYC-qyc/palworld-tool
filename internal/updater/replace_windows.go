//go:build windows

package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// applyUpdate 在 Windows 上解压 zip 并通过 .bat 脚本替换文件、重启面板。
func applyUpdate(asset, tmpZip, installDir, service string, onProgress func(Progress)) error {
	// 解压到临时目录
	tmpDir := filepath.Join(os.TempDir(), "paladmin-update-"+time.Now().Format("20060102150405"))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}

	powershell := fmt.Sprintf("Expand-Archive -Force -Path '%s' -DestinationPath '%s'", tmpZip, tmpDir)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", powershell)
	cmd.SysProcAttr = newSysProcAttr()
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("解压失败: %w: %s", err, string(out))
	}

	onProgress(Progress{Stage: "restart", Message: "正在替换文件并重启...", Percent: 97})

	// 替换脚本：等待当前进程退出，然后复制文件、启动新版本、自删除
	batPath := filepath.Join(os.TempDir(), "paladmin-update-"+time.Now().Format("150405")+".bat")
	batContent := fmt.Sprintf(`@echo off
ping 127.0.0.1 -n 3 > nul
taskkill /F /IM paladmin.exe 2>nul
xcopy /E /Y /I "%s\*" "%s\"
start "" "%s\paladmin.exe"
del "%s"
`, tmpDir, installDir, installDir, batPath)
	if err := os.WriteFile(batPath, []byte(batContent), 0755); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("写入更新脚本失败: %w", err)
	}

	cmd = exec.Command("cmd", "/C", "start", "/B", batPath)
	cmd.SysProcAttr = newSysProcAttr()
	if err := cmd.Start(); err != nil {
		os.RemoveAll(tmpDir)
		_ = os.Remove(batPath)
		return fmt.Errorf("启动更新脚本失败: %w", err)
	}

	return nil
}
