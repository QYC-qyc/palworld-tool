package task

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/spf13/viper"
	"go.etcd.io/bbolt"
	"palworld-panel/internal/database"
	"palworld-panel/internal/gamesrv"
	"palworld-panel/internal/logger"
	"palworld-panel/internal/tool"
	"palworld-panel/service"
)

var (
	scheduler gocron.Scheduler
	dbRef     *bbolt.DB

	playerCache = map[string]string{}
	firstPoll   = true
)

// Init 设置依赖
func Init(db *bbolt.DB) {
	dbRef = db
}

// SyncPlayersOnce 执行一次在线玩家同步
func SyncPlayersOnce() {
	if dbRef == nil {
		return
	}
	online, err := tool.ShowPlayers()
	if err != nil {
		logger.Errorf("同步在线玩家失败: %v", err)
		return
	}
	if err := service.PutPlayersOnline(dbRef, online); err != nil {
		logger.Errorf("写入在线玩家失败: %v", err)
	}

	if viper.GetBool("task.player_logging") {
		playerLogging(online)
	}
	if viper.GetBool("manage.kick_non_whitelist") {
		checkAndKickPlayers(online)
	}
}

// SyncSavOnce 执行一次存档同步
func SyncSavOnce() error {
	return tool.Decode(tool.EffectiveSavePath())
}

func backupTask() {
	// 游戏未启动时不备份（Level.sav 可能不存在或未更新）
	if !isGameRunning() {
		logger.Debugf("游戏服未运行，跳过本次备份")
		return
	}
	path, err := tool.Backup()
	if err != nil {
		logger.Errorf("备份失败: %v", err)
		return
	}
	_ = service.AddBackup(dbRef, database.Backup{Path: path})
	logger.Infof("自动备份完成: %s", path)
	// 按保留策略清理旧备份
	if removed, err := service.PruneBackups(dbRef,
		viper.GetInt("backup.keep_count"),
		viper.GetInt("backup.keep_days")); err != nil {
		logger.Warnf("清理旧备份失败: %v", err)
	} else if removed > 0 {
		logger.Infof("已清理 %d 个过期备份", removed)
	}
}

// isGameRunning 检查游戏服是否在运行。
// Docker 部署下通过 Manager 查询容器状态；本地部署回退到 pgrep/tasklist。
func isGameRunning() bool {
	if gamesrv.Default != nil {
		st, err := gamesrv.Default.GetStatus()
		if err != nil {
			return false
		}
		return st.Running
	}
	const exeName = "PalServer-Win64-Shipping-Cmd.exe"
	if runtime.GOOS == "windows" {
		out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq "+exeName, "/FO", "CSV", "/NH").Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(out), exeName)
	}
	out, err := exec.Command("pgrep", "-f", exeName).Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

func playerLogging(players []database.OnlinePlayer) {
	loginMsg := viper.GetString("task.player_login_message")
	logoutMsg := viper.GetString("task.player_logout_message")
	tmp := map[string]string{}
	for _, p := range players {
		if p.PlayerUid != "" {
			tmp[p.PlayerUid] = p.Nickname
		}
	}
	if !firstPoll {
		for id, name := range tmp {
			if _, ok := playerCache[id]; !ok {
				broadcastVariables(loginMsg, name, len(players))
			}
		}
		for id, name := range playerCache {
			if _, ok := tmp[id]; !ok {
				broadcastVariables(logoutMsg, name, len(players))
			}
		}
	}
	firstPoll = false
	playerCache = tmp
}

func broadcastVariables(message, username string, online int) {
	message = strings.ReplaceAll(message, "{username}", username)
	message = strings.ReplaceAll(message, "{online_num}", strconv.Itoa(online))
	for _, line := range strings.Split(message, "\n") {
		if line == "" {
			continue
		}
		if err := tool.Broadcast(line); err != nil {
			logger.Warnf("广播失败: %v", err)
		}
		time.Sleep(time.Second)
	}
}

func checkAndKickPlayers(players []database.OnlinePlayer) {
	whitelist, err := service.ListWhitelist(dbRef)
	if err != nil {
		return
	}
	for _, p := range players {
		if p.SteamId == "" {
			continue
		}
		whitelisted := false
		for _, w := range whitelist {
			if (p.PlayerUid != "" && p.PlayerUid == w.PlayerUID) || (p.SteamId == w.SteamID) {
				whitelisted = true
				break
			}
		}
		if !whitelisted {
			if err := tool.KickPlayer(p.SteamId); err != nil {
				logger.Warnf("踢出非白名单玩家 %s 失败: %v", p.Nickname, err)
			}
		}
	}
}

// Schedule 启动定时任务
func Schedule() error {
	s, err := gocron.NewScheduler()
	if err != nil {
		return err
	}
	scheduler = s

	if interval := viper.GetInt("task.sync_interval"); interval > 0 {
		go SyncPlayersOnce()
		if _, err := s.NewJob(gocron.DurationJob(time.Duration(interval)*time.Second),
			gocron.NewTask(SyncPlayersOnce)); err != nil {
			logger.Errorf("注册在线同步任务失败: %v", err)
		}
	}
	if interval := viper.GetInt("save.sync_interval"); interval > 0 {
		go func() { _ = SyncSavOnce() }()
		if _, err := s.NewJob(gocron.DurationJob(time.Duration(interval)*time.Second),
			gocron.NewTask(func() { _ = SyncSavOnce() })); err != nil {
			logger.Errorf("注册存档同步任务失败: %v", err)
		}
	}
	if interval := viper.GetInt("save.backup_interval"); interval > 0 {
		go backupTask()
		if _, err := s.NewJob(gocron.DurationJob(time.Duration(interval)*time.Second),
			gocron.NewTask(backupTask)); err != nil {
			logger.Errorf("注册备份任务失败: %v", err)
		}
	}
	s.Start()
	return nil
}

// Shutdown 停止调度器
func Shutdown() {
	if scheduler != nil {
		_ = scheduler.Shutdown()
	}
}
