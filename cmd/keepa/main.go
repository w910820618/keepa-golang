package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"keepa/internal/config"
	"keepa/internal/database"
	"keepa/internal/logger"
	"keepa/internal/scheduler"
	"keepa/internal/task"
	"keepa/internal/tasks"

	"go.uber.org/zap"
)

func main() {
	// 加载配置
	cfg, err := config.Load("")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	logCfg := logger.LoggerConfig{
		Level:      cfg.Logger.Level,
		Format:     cfg.Logger.Format,
		OutputPath: cfg.Logger.OutputPath,
		MaxSize:    cfg.Logger.MaxSize,
		MaxBackups: cfg.Logger.MaxBackups,
		MaxAge:     cfg.Logger.MaxAge,
		Compress:   cfg.Logger.Compress,
	}

	zapLogger, err := logger.NewLogger(logCfg)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer zapLogger.Sync()

	zapLogger.Info("application starting",
		zap.String("name", cfg.App.Name),
		zap.String("version", cfg.App.Version),
		zap.String("env", cfg.App.Env),
	)

	// 初始化数据库连接
	dbCfg := database.Config{
		MySQL: database.MySQLConfig{
			Enabled:         cfg.Database.MySQL.Enabled,
			Host:            cfg.Database.MySQL.Host,
			Port:            cfg.Database.MySQL.Port,
			Username:        cfg.Database.MySQL.Username,
			Password:        cfg.Database.MySQL.Password,
			Database:        cfg.Database.MySQL.Database,
			Charset:         cfg.Database.MySQL.Charset,
			MaxOpenConns:    cfg.Database.MySQL.MaxOpenConns,
			MaxIdleConns:    cfg.Database.MySQL.MaxIdleConns,
			ConnMaxLifetime: cfg.Database.MySQL.ConnMaxLifetime,
			ConnMaxIdleTime: cfg.Database.MySQL.ConnMaxIdleTime,
		},
		PostgreSQL: database.PostgreSQLConfig{
			Enabled:         cfg.Database.PostgreSQL.Enabled,
			Host:            cfg.Database.PostgreSQL.Host,
			Port:            cfg.Database.PostgreSQL.Port,
			Username:        cfg.Database.PostgreSQL.Username,
			Password:        cfg.Database.PostgreSQL.Password,
			Database:        cfg.Database.PostgreSQL.Database,
			SSLMode:         cfg.Database.PostgreSQL.SSLMode,
			MaxOpenConns:    cfg.Database.PostgreSQL.MaxOpenConns,
			MaxIdleConns:    cfg.Database.PostgreSQL.MaxIdleConns,
			ConnMaxLifetime: cfg.Database.PostgreSQL.ConnMaxLifetime,
			ConnMaxIdleTime: cfg.Database.PostgreSQL.ConnMaxIdleTime,
		},
		MongoDB: database.MongoDBConfig{
			Enabled:     cfg.Database.MongoDB.Enabled,
			URI:         cfg.Database.MongoDB.URI,
			Database:    cfg.Database.MongoDB.Database,
			AuthSource:  cfg.Database.MongoDB.AuthSource,
			Username:    cfg.Database.MongoDB.Username,
			Password:    cfg.Database.MongoDB.Password,
			ReplicaSet:  cfg.Database.MongoDB.ReplicaSet,
			MaxPoolSize: cfg.Database.MongoDB.MaxPoolSize,
			MinPoolSize: cfg.Database.MongoDB.MinPoolSize,
			MaxIdleTime: cfg.Database.MongoDB.MaxIdleTime,
		},
		Logger: zapLogger,
	}

	dbs, err := database.New(dbCfg)
	if err != nil {
		zapLogger.Fatal("failed to initialize databases", zap.Error(err))
	}

	// 设置全局数据库管理器
	database.SetGlobal(dbs)

	// 创建任务注册表
	registry := task.NewTaskRegistry()

	// 注册任务
	if err := registerTasks(registry); err != nil {
		zapLogger.Fatal("failed to register tasks", zap.Error(err))
	}

	// 获取时区和超时配置
	location, err := cfg.GetLocation()
	if err != nil {
		zapLogger.Warn("failed to load location, using local time",
			zap.Error(err),
		)
		location = time.Local
	}

	defaultTimeout, err := cfg.GetDefaultTimeout()
	if err != nil {
		zapLogger.Warn("failed to parse default timeout, using 30m",
			zap.Error(err),
		)
		defaultTimeout = 30 * time.Minute
	}

	// 创建调度器
	sched := scheduler.NewScheduler(scheduler.Config{
		Logger:         zapLogger,
		Registry:       registry,
		DefaultTimeout: defaultTimeout,
		Location:       location,
	})

	// 启动调度器
	if err := sched.Start(); err != nil {
		zapLogger.Fatal("failed to start scheduler", zap.Error(err))
	}

	zapLogger.Info("scheduler started successfully",
		zap.Int("task_count", sched.GetTaskCount()),
	)

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	zapLogger.Info("received signal, shutting down...",
		zap.String("signal", sig.String()),
	)

	// 优雅关闭
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := sched.Stop(shutdownCtx); err != nil {
		zapLogger.Error("error stopping scheduler", zap.Error(err))
	}

	// 关闭数据库连接
	if err := dbs.Close(); err != nil {
		zapLogger.Error("error closing databases", zap.Error(err))
	}

	zapLogger.Info("application stopped")
}

// registerTasks 注册所有任务
func registerTasks(registry *task.TaskRegistry) error {
	// 注册示例任务
	if err := registry.Register(tasks.NewExampleTask()); err != nil {
		return fmt.Errorf("failed to register example task: %w", err)
	}

	// 注册清理任务（示例）
	// 注意：在实际使用中，您可以根据配置决定是否启用某些任务
	if err := registry.Register(tasks.NewCleanupTask()); err != nil {
		return fmt.Errorf("failed to register cleanup task: %w", err)
	}

	// 在这里注册更多任务...

	return nil
}
