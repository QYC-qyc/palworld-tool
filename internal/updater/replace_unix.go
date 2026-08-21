//go:build !windows

package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// applyUpdate 在 Unix 上解压 tar.gz 并通过 bash 脚本替换文件、重启服务。
func applyUpdate(asset, tmpTar, installDir, service string, onProgress func(Progress)) error {
	// 先解压到临时目录（不能 defer 删除，替换脚本还要用）
	tmpDir := filepath.Join(os.TempDir(), "palworld-panel-update-"+time.Now().Format("20060102150405"))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}

	cmd := exec.Command("tar", "-xzf", tmpTar, "-C", tmpDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("解压失败: %w: %s", err, string(out))
	}

	// 设置权限
	_ = os.Chmod(filepath.Join(tmpDir, "palworld-panel"), 0755)
	if sav := filepath.Join(tmpDir, "sav_cli"); fileExists(sav) {
		_ = os.Chmod(sav, 0755)
	}

	onProgress(Progress{Stage: "restart", Message: "正在替换文件并重启...", Percent: 97})
	// 替换脚本：sleep 等待当前进程退出，然后复制文件、清理临时目录、重启服务
	replaceScript := fmt.Sprintf(`sleep 2
cp -rf %s/* %s/
chmod +x %s/palworld-panel
[ -f %s/sav_cli ] && chmod +x %s/sav_cli
rm -rf %s
systemctl restart %s
`, tmpDir, installDir, installDir, tmpDir, tmpDir, tmpDir, service)
	cmd = exec.Command("setsid", "bash", "-c", replaceScript)
	cmd.SysProcAttr = newSysProcAttr()
	if err := cmd.Start(); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("启动替换脚本失败: %w", err)
	}

	return nil
}
