package database

import (
	"keepa/internal/config"
	
	"go.uber.org/zap"
)

// ConfigFromAppConfig 从应用配置转换为数据库配置
func ConfigFromAppConfig(cfg *config.Config, logger *zap.Logger) Config {
	return Config{
		MySQL: MySQLConfig{
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
		PostgreSQL: PostgreSQLConfig{
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
		MongoDB: MongoDBConfig{
			Enabled:    cfg.Database.MongoDB.Enabled,
			URI:        cfg.Database.MongoDB.URI,
			Database:   cfg.Database.MongoDB.Database,
			AuthSource: cfg.Database.MongoDB.AuthSource,
			Username:   cfg.Database.MongoDB.Username,
			Password:   cfg.Database.MongoDB.Password,
			ReplicaSet: cfg.Database.MongoDB.ReplicaSet,
			MaxPoolSize: cfg.Database.MongoDB.MaxPoolSize,
			MinPoolSize: cfg.Database.MongoDB.MinPoolSize,
			MaxIdleTime: cfg.Database.MongoDB.MaxIdleTime,
		},
		Logger: logger,
	}
}

