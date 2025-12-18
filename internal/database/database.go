package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// Databases 数据库连接管理器
type Databases struct {
	MySQL      *sql.DB
	PostgreSQL *sql.DB
	MongoDB    *mongo.Database
	logger     *zap.Logger
}

// Config 数据库配置
type Config struct {
	MySQL      MySQLConfig
	PostgreSQL PostgreSQLConfig
	MongoDB    MongoDBConfig
	Logger     *zap.Logger
}

// MySQLConfig MySQL 配置
type MySQLConfig struct {
	Enabled         bool
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	Charset         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// PostgreSQLConfig PostgreSQL 配置
type PostgreSQLConfig struct {
	Enabled         bool
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// MongoDBConfig MongoDB 配置
type MongoDBConfig struct {
	Enabled    bool
	URI        string
	Database   string
	AuthSource string
	Username   string
	Password   string
	ReplicaSet string
	MaxPoolSize uint64
	MinPoolSize uint64
	MaxIdleTime string
}

// New 创建数据库连接管理器
func New(cfg Config) (*Databases, error) {
	db := &Databases{
		logger: cfg.Logger,
	}
	
	// 连接 MySQL
	if cfg.MySQL.Enabled {
		if err := db.connectMySQL(cfg.MySQL); err != nil {
			return nil, fmt.Errorf("failed to connect MySQL: %w", err)
		}
		if cfg.Logger != nil {
			cfg.Logger.Info("MySQL connected successfully")
		}
	}
	
	// 连接 PostgreSQL
	if cfg.PostgreSQL.Enabled {
		if err := db.connectPostgreSQL(cfg.PostgreSQL); err != nil {
			return nil, fmt.Errorf("failed to connect PostgreSQL: %w", err)
		}
		if cfg.Logger != nil {
			cfg.Logger.Info("PostgreSQL connected successfully")
		}
	}
	
	// 连接 MongoDB
	if cfg.MongoDB.Enabled {
		if err := db.connectMongoDB(cfg.MongoDB); err != nil {
			return nil, fmt.Errorf("failed to connect MongoDB: %w", err)
		}
		if cfg.Logger != nil {
			cfg.Logger.Info("MongoDB connected successfully")
		}
	}
	
	return db, nil
}

// connectMySQL 连接 MySQL
func (d *Databases) connectMySQL(cfg MySQLConfig) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.Charset,
	)
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open MySQL connection: %w", err)
	}
	
	// 设置连接池参数
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	
	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping MySQL: %w", err)
	}
	
	d.MySQL = db
	return nil
}

// connectPostgreSQL 连接 PostgreSQL
func (d *Databases) connectPostgreSQL(cfg PostgreSQLConfig) error {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.Username,
		cfg.Password,
		cfg.Database,
		cfg.SSLMode,
	)
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}
	
	// 设置连接池参数
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	
	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}
	
	d.PostgreSQL = db
	return nil
}

// connectMongoDB 连接 MongoDB
func (d *Databases) connectMongoDB(cfg MongoDBConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// 构建连接选项
	opts := options.Client().ApplyURI(cfg.URI)
	
	// 如果提供了用户名和密码，设置认证
	if cfg.Username != "" && cfg.Password != "" {
		credential := options.Credential{
			AuthSource: cfg.AuthSource,
			Username:   cfg.Username,
			Password:   cfg.Password,
		}
		opts.SetAuth(credential)
	}
	
	// 设置连接池参数
	if cfg.MaxPoolSize > 0 {
		opts.SetMaxPoolSize(cfg.MaxPoolSize)
	}
	if cfg.MinPoolSize > 0 {
		opts.SetMinPoolSize(cfg.MinPoolSize)
	}
	if cfg.MaxIdleTime != "" {
		if maxIdleTime, err := time.ParseDuration(cfg.MaxIdleTime); err == nil {
			opts.SetMaxConnIdleTime(maxIdleTime)
		}
	}
	
	// 连接 MongoDB
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to connect MongoDB: %w", err)
	}
	
	// 测试连接
	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}
	
	// 获取数据库
	d.MongoDB = client.Database(cfg.Database)
	return nil
}

// Close 关闭所有数据库连接
func (d *Databases) Close() error {
	var errs []error
	
	if d.MySQL != nil {
		if err := d.MySQL.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close MySQL: %w", err))
		} else if d.logger != nil {
			d.logger.Info("MySQL connection closed")
		}
	}
	
	if d.PostgreSQL != nil {
		if err := d.PostgreSQL.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close PostgreSQL: %w", err))
		} else if d.logger != nil {
			d.logger.Info("PostgreSQL connection closed")
		}
	}
	
	if d.MongoDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := d.MongoDB.Client().Disconnect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to close MongoDB: %w", err))
		} else if d.logger != nil {
			d.logger.Info("MongoDB connection closed")
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("errors closing databases: %v", errs)
	}
	
	return nil
}

// Ping 检查所有已启用数据库的连接状态
func (d *Databases) Ping(ctx context.Context) error {
	if d.MySQL != nil {
		if err := d.MySQL.PingContext(ctx); err != nil {
			return fmt.Errorf("MySQL ping failed: %w", err)
		}
	}
	
	if d.PostgreSQL != nil {
		if err := d.PostgreSQL.PingContext(ctx); err != nil {
			return fmt.Errorf("PostgreSQL ping failed: %w", err)
		}
	}
	
	if d.MongoDB != nil {
		if err := d.MongoDB.Client().Ping(ctx, nil); err != nil {
			return fmt.Errorf("MongoDB ping failed: %w", err)
		}
	}
	
	return nil
}

