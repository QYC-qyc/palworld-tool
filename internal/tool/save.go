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

// getFromSource 根据 path 前缀获取存档，目前支持本地路径，预留 http/docker/k8s
func getFromSource(path, way string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		// agent 模式：HTTP 下载 zip（后续实现）
		return "", errors.New("HTTP 存档来源暂未实现，请使用本地路径")
	}
	return source.CopyFromLocal(path, way)
}

// Decode 调用 sav_cli 解析 Level.sav，结果通过 HTTP 回写
func Decode(path string) error {
	savCli, err := getSavCli()
	if err != nil {
		return errors.New("获取 sav_cli 失败: " + err.Error())
	}

	levelFile, err := getFromSource(path, "decode")
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(levelFile))

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

// EffectiveSavePath 返回实际使用的存档目录：
// 优先使用用户配置的 save.path；为空时从游戏安装目录自动推导
// （<install_dir>/Pal/Saved）。
func EffectiveSavePath() string {
	if p := viper.GetString("save.path"); p != "" {
		return p
	}
	if installDir := viper.GetString("gamesrv.install_dir"); installDir != "" {
		return filepath.Join(installDir, "Pal", "Saved")
	}
	return ""
}

// Backup 备份存档为 zip，返回文件名
func Backup() (string, error) {
	sourcePath := EffectiveSavePath()
	levelFile, err := getFromSource(sourcePath, "backup")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(filepath.Dir(levelFile))

	backupDir := filepath.Join(mustWd(), "backups")
	if err := system.CheckAndCreateDir(backupDir); err != nil {
		return "", err
	}
	name := time.Now().Format("2006-01-02-15-04-05") + ".zip"
	backupZip := filepath.Join(backupDir, name)
	if err := system.ZipDir(filepath.Dir(levelFile), backupZip); err != nil {
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
