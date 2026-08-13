package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"paladmin/api"
	"paladmin/internal/config"
	"paladmin/internal/database"
	"paladmin/internal/logger"
	"paladmin/internal/system"
	"paladmin/internal/task"
	"paladmin/service"
	"paladmin/service/anticheat"

	"go.etcd.io/bbolt"
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
	db := database.GetDB(dbPath)
	defer database.Close()

	// 初始化运行时可改配置：首次从 config.yaml 写入数据库，之后以数据库为准
	initRuntimeSettings(db)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(func(c *gin.Context) {
		c.Set("version", version)
		c.Next()
	})

	engine := anticheat.New(db, &conf.Anticheat)
	api.SetDeps(db, engine, &conf)
	api.RegisterRouter(router)

	// 静态前端：自动查找 web/dist 或 web 目录
	webDir := ""
	for _, d := range []string{"web/dist", "web"} {
		if _, err := os.Stat(filepath.Join(d, "index.html")); err == nil {
			webDir = d
			break
		}
	}
	if webDir != "" {
		router.StaticFS("/assets", http.Dir(filepath.Join(webDir, "assets")))
		// SPA 入口与前端路由 fallback
		router.NoRoute(func(c *gin.Context) {
			// API 路径不返回 index.html
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

	task.Init(db, engine)

	localIP, _ := system.GetLocalIP()
	port := viper.GetInt("web.port")
	logger.Info("PalAdmin 启动中...")
	logger.Infof("版本: %s", version)
	logger.Infof("监听: http://127.0.0.1:%d 或 http://%s:%d", port, localIP, port)

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

// initRuntimeSettings 首次把 config.yaml 的值同步进数据库 settings，
// 并将数据库中的值应用到 viper，使后续读取以面板设置为准。
func initRuntimeSettings(db *bbolt.DB) {
	defaults := service.DefaultSettings()
	// 用 config.yaml/env 中的连接配置作为首次写入的默认值，
	// 但 web.password 除外——它必须由首次访问面板的初始化向导设置。
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
