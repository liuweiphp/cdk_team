package main

import (
	"embed"
	"exchange_cdk/config"
	"exchange_cdk/middleware"
	pkgmigrate "exchange_cdk/pkg/migrate"
	"exchange_cdk/router"
	"exchange_cdk/service"
	"fmt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

//go:embed migration/*.sql
var migrationsFS embed.FS

func main() {
	cfg := config.Load()

	// 初始化日志
	logger := initLogger(cfg.LogLevel)
	defer logger.Sync()

	// 自动执行未跑过的数据库迁移(幂等,已是最新则跳过)
	if err := pkgmigrate.Run(cfg.DBDSN, migrationsFS, "migration", logger); err != nil {
		logger.Fatal("数据库迁移失败", zap.Error(err))
	}

	// 初始化数据库
	db, err := gorm.Open(mysql.Open(cfg.DBDSN), &gorm.Config{})
	if err != nil {
		logger.Fatal("数据库连接失败", zap.Error(err))
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)

	runner := service.NewAutomationRunner(
		cfg.AutomationPythonBin,
		cfg.AutomationScriptPath,
		cfg.AutomationTimeoutSeconds,
		cfg.AutomationMaxRetries,
	)

	// 初始化服务
	svc := &router.Services{
		Auth:         service.NewAuthService(db, cfg.JWTSecret, cfg.BcryptCost),
		User:         service.NewUserService(db, cfg.BcryptCost),
		Cdk:          service.NewCdkService(db),
		Exchange:     service.NewExchangeService(db, cfg.MaxExchangeQty),
		Redeem:       service.NewRedeemService(db),
		RedeemItem:   service.NewRedeemItemService(db),
		Template:     service.NewTemplateService(db),
		Team:         service.NewTeamService(db),
		PurchaseTask: service.NewPurchaseTaskService(db, runner),
		Announcement: service.NewAnnouncementService(db),
		Stats:        service.NewStatsService(db),
	}
	svc.RedeemItem.SetPurchaseTaskService(svc.PurchaseTask)

	// 初始化路由
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger(logger))

	router.Setup(r, svc, cfg, logger, db)

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	logger.Info("服务器启动", zap.String("addr", addr))
	if err := r.Run(addr); err != nil {
		logger.Fatal("服务器启动失败", zap.Error(err))
	}
}

func initLogger(level string) *zap.Logger {
	var lvl zapcore.Level
	switch level {
	case "debug":
		lvl = zapcore.DebugLevel
	case "warn":
		lvl = zapcore.WarnLevel
	case "error":
		lvl = zapcore.ErrorLevel
	default:
		lvl = zapcore.InfoLevel
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.Encoding = "json"
	logger, _ := cfg.Build()
	return logger
}
