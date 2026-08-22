package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"palworld-panel/api"
	"palworld-panel/internal/compose"
	"palworld-panel/internal/config"
	"palworld-panel/internal/database"
	"palworld-panel/internal/logger"
	"palworld-panel/internal/system"
	"palworld-panel/internal/task"
	"palworld-panel/internal/updater"
	"palworld-panel/internal/webdata"
	"palworld-panel/service"
)

var (
	version = "dev"
	cfgFile string
	conf    config.Config
)

func setupFlags() {
	flag.StringVar(&cfgFile, "config", "", "配置文件路径")
	flag.Parse()
}

func main() {
	setupFlags()
	config.Init(cfgFile, &conf)
	logger.Init(conf.Log.Level, conf.Log.File)

	dbPath := viper.GetString("storage.path")
	if dbPath == "" {
		dbPath = "./pst.db"
	}
	_ = os.MkdirAll(filepath.Dir(dbPath), 0755)
	db := database.InitDB(dbPath)
	defer db.Close()

	// 初始化运行时可改配置：首次从 config.yaml 写入数据库，之后以数据库为准
	initRuntimeSettings(db)

	// 容器模式：确保 /compose/docker-compose.yml 存在（不存在则释放内置模板）
	ensureComposeFile()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(func(c *gin.Context) {
		c.Set("version", version)
		c.Next()
	})

	api.SetDeps(db, &conf)
	api.RegisterRouter(router)

	// 静态前端：优先从 web.dat 释放（发布版，避免上万个小文件解压慢）；
	// 开发环境回退到 web/dist 或 web 目录。
	webDir := ""
	dataDir := getDataDir()
	if released, err := webdata.EnsureExtracted(dataDir, version, ""); err == nil {
		webDir = released
		logger.Infof("前端资源（已释放）: %s", webDir)
	} else {
		for _, d := range []string{"web/dist", "web"} {
			if _, err := os.Stat(filepath.Join(d, "index.html")); err == nil {
				webDir = d
				break
			}
		}
	}
	if webDir != "" {
		router.StaticFS("/assets", http.Dir(filepath.Join(webDir, "assets")))
		router.StaticFS("/data", http.Dir(filepath.Join(webDir, "data")))
		router.StaticFS("/icons", http.Dir(filepath.Join(webDir, "icons")))
		router.StaticFS("/map", http.Dir(filepath.Join(webDir, "map")))
		router.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.File(filepath.Join(webDir, "index.html"))
		})
		logger.Infof("前端目录: %s", webDir)
	} else {
		logger.Warn("未找到前端资源（web/dist 或 web），仅提供 API")
	}

	task.Init(db)

	localIP, _ := system.GetLocalIP()
	port := viper.GetInt("web.port")
	updater.SetVersion(version)
	logger.Info("PalAdmin 启动中...")
	logger.Infof("版本: %s", version)
	logger.Infof("监听: http://127.0.0.1:%d 或 http://%s:%d", port, localIP, port)

	// Windows 下打印醒目提示并尝试自动打开浏览器
	if runtime.GOOS == "windows" {
		url := fmt.Sprintf("http://127.0.0.1:%d", port)
		fmt.Println()
		fmt.Println("  ==========================================")
		fmt.Println("   PalAdmin 已启动！")
		fmt.Printf("   请在浏览器中打开：%s\n", url)
		fmt.Println("  ==========================================")
		fmt.Println()
		// 延迟 1.5 秒后自动打开默认浏览器
		go func() {
			time.Sleep(1500 * time.Millisecond)
			_ = exec.Command("cmd", "/c", "start", "", url).Start()
		}()
	}

	if err := task.Schedule(); err != nil {
		logger.Errorf("启动定时任务失败: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		var err error
		if viper.GetBool("web.tls") {
			err = router.RunTLS(fmt.Sprintf(":%d", port),
				viper.GetString("web.cert_path"),
				viper.GetString("web.key_path"))
		} else {
			err = router.Run(fmt.Sprintf(":%d", port))
		}
		if err != nil {
			logger.Errorf("HTTP 服务退出: %v", err)
		}
	}()

	<-sigChan
	task.Shutdown()
	logger.Info("已优雅停止")
}

func initRuntimeSettings(db *database.Store) {
	defaults := service.DefaultSettings()
	for k := range defaults {
		if k == service.SettingWebPassword {
			continue
		}
		if v := viper.GetString(k); v != "" {
			defaults[k] = v
		}
	}
	_ = service.InitSettings(db, defaults)

	saved, err := service.GetAllSettings(db)
	if err == nil {
		config.ApplyToViper(saved)
		logger.Infof("已加载 %d 项动态配置", len(saved))
	}
}

// getDataDir 返回用户数据目录（前端资源释放、数据库等）。
// Windows: %APPDATA%\PalAdmin；其他系统：可执行文件所在目录。
func getDataDir() string {
	if d := os.Getenv("APPDATA"); d != "" {
		dir := filepath.Join(d, "PalAdmin")
		_ = os.MkdirAll(dir, 0o755)
		return dir
	}
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	return "."
}

// ensureComposeFile 如果 COMPOSE_PROJECT_DIR 下的 docker-compose.yml 不存在，
// 释放内置模板，避免用户手动下载。仅在容器内（挂载了 /compose）有意义。
func ensureComposeFile() {
	dir := os.Getenv("COMPOSE_PROJECT_DIR")
	if dir == "" {
		return
	}
	path := filepath.Join(dir, "docker-compose.yml")
	if _, err := os.Stat(path); err == nil {
		return // 已存在，不覆盖
	}
	_ = os.MkdirAll(dir, 0o755)
	tmpl := compose.Template()
	if len(tmpl) == 0 {
		return
	}
	if err := os.WriteFile(path, tmpl, 0o644); err != nil {
		logger.Warnf("释放 compose 模板失败: %v", err)
	} else {
		logger.Infof("已释放 docker-compose.yml 到 %s", path)
	}
}
