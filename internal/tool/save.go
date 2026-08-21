package tool

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/viper"
	"paladmin/internal/auth"
	"paladmin/internal/gamesrv"
	"paladmin/internal/logger"
	"paladmin/internal/source"
	"paladmin/internal/system"
)

type Structure struct {
	Players []interface{} `json:"players"`
	Guilds  []interface{} `json:"guilds"`
}

func getSavCli() (string, error) {
	savCliPath := viper.GetString("save.decode_path")
	if savCliPath == "" || strings.Contains(savCliPath, "/path/to/") {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		savCliPath = filepath.Join(wd, "sav_cli")
		if runtime.GOOS == "windows" {
			savCliPath += ".exe"
		}
	}
	if _, err := os.Stat(savCliPath); err != nil {
		return "", err
	}
	return savCliPath, nil
}

// fetchSavedToLocal 通过游戏文件访问层把游戏 Saved 目录拉到本地临时目录，
// 返回其中 Level.sav 的本地路径，以及清理函数。
func fetchSavedToLocal(way string) (levelFile string, cleanup func(), err error) {
	tmpRoot, err := os.MkdirTemp("", "paladm-"+way+"-")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { _ = os.RemoveAll(tmpRoot) }
	if gamesrv.Default != nil {
		if err := gamesrv.Default.FetchSavedToLocal(tmpRoot); err != nil {
			cleanup()
			return "", nil, err
		}
	} else {
		// 兜底：直接从本地路径拷贝（非容器、且未注入 manager 的场景）
		savedDir := EffectiveSavePath()
		if _, err := os.Stat(savedDir); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("存档目录不可用: %w", err)
		}
		if err := source.CopySavedTree(savedDir, filepath.Join(tmpRoot, "Pal", "Saved")); err != nil {
			cleanup()
			return "", nil, err
		}
	}
	level, err := source.FindLevelSav(filepath.Join(tmpRoot, "Pal", "Saved"))
	if err != nil {
		cleanup()
		return "", nil, err
	}
	return level, cleanup, nil
}

// Decode 调用 sav_cli 解析 Level.sav，结果通过 HTTP 回写
func Decode(path string) error {
	savCli, err := getSavCli()
	if err != nil {
		return errors.New("获取 sav_cli 失败: " + err.Error())
	}

	levelFile, cleanup, err := fetchSavedToLocal("decode")
	if err != nil {
		return err
	}
	defer cleanup()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", viper.GetInt("web.port"))
	if viper.GetBool("web.tls") && viper.GetString("web.public_url") != "" {
		baseURL = viper.GetString("web.public_url")
	}
	requestURL := baseURL + "/api/"
	token, err := auth.GenerateToken()
	if err != nil {
		return errors.New("生成 token 失败: " + err.Error())
	}

	args := []string{"-f", levelFile, "--request", requestURL, "--token", token}
	cmd := exec.Command(savCli, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return errors.New("启动 sav_cli 失败: " + err.Error())
	}
	return cmd.Wait()
}

// EffectiveSavePath 返回存档目录路径（仅用于本地模式/展示；容器模式下文件访问走 GameFS）。
func EffectiveSavePath() string {
	if p := viper.GetString("save.path"); p != "" {
		return p
	}
	if installDir := viper.GetString("gamesrv.install_dir"); installDir != "" {
		return filepath.Join(installDir, "Pal", "Saved")
	}
	return ""
}

// BackupDir 返回备份 zip 存放目录（位于持久化数据目录下）。
func BackupDir() string {
	// 优先用 PALADIN_DATA_DIR（容器入口脚本设置为 /data）
	if d := os.Getenv("PALADIN_DATA_DIR"); d != "" {
		return filepath.Join(d, "backups")
	}
	// 其次与数据库同目录
	if db := viper.GetString("storage.path"); db != "" && db != "./pst.db" {
		return filepath.Join(filepath.Dir(db), "backups")
	}
	return filepath.Join(mustWd(), "backups")
}

// Backup 通过游戏文件访问层拉取存档并打包为 zip，返回文件名
func Backup() (string, error) {
	tmpRoot, err := os.MkdirTemp("", "paladm-backup-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpRoot)

	// 拉取游戏 Saved 目录到 tmpRoot/Pal/Saved
	if gamesrv.Default != nil {
		if err := gamesrv.Default.FetchSavedToLocal(tmpRoot); err != nil {
			return "", err
		}
	} else {
		savedDir := EffectiveSavePath()
		if err := source.CopySavedTree(savedDir, filepath.Join(tmpRoot, "Pal", "Saved")); err != nil {
			return "", err
		}
	}

	backupDir := BackupDir()
	if err := system.CheckAndCreateDir(backupDir); err != nil {
		return "", err
	}
	name := time.Now().Format("2006-01-02-15-04-05") + ".zip"
	backupZip := filepath.Join(backupDir, name)
	// 打包 tmpRoot 下的 Pal/ 目录（zip 内顶层为 Saved/）
	if err := system.ZipDir(filepath.Join(tmpRoot, "Pal"), backupZip); err != nil {
		return "", fmt.Errorf("创建备份失败: %s", err)
	}
	logger.Infof("存档已备份到 %s", backupZip)
	return name, nil
}

func mustWd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
