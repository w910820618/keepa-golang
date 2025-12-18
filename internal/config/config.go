package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config 应用程序配置
type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
	Logger    LoggerConfig    `mapstructure:"logger"`
	Database  DatabaseConfig  `mapstructure:"database"`
	KeepaAPI  KeepaAPIConfig  `mapstructure:"keepa_api"`
	Server    ServerConfig    `mapstructure:"server"`
}

// AppConfig 应用基础配置
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Env     string `mapstructure:"env"` // development, production
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	DefaultTimeout string `mapstructure:"default_timeout"` // 例如: "30m"
	Location       string `mapstructure:"location"`        // 例如: "Asia/Shanghai"
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	Level      string `mapstructure:"level"`       // debug, info, warn, error
	Format     string `mapstructure:"format"`      // json, console
	OutputPath string `mapstructure:"output_path"` // 日志输出路径
	MaxSize    int    `mapstructure:"max_size"`    // 日志文件最大大小(MB)
	MaxBackups int    `mapstructure:"max_backups"` // 保留的日志文件数量
	MaxAge     int    `mapstructure:"max_age"`     // 日志保留天数
	Compress   bool   `mapstructure:"compress"`    // 是否压缩旧日志
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	MySQL      MySQLConfig      `mapstructure:"mysql"`
	PostgreSQL PostgreSQLConfig `mapstructure:"postgresql"`
	MongoDB    MongoDBConfig    `mapstructure:"mongodb"`
}

// MySQLConfig MySQL 配置
type MySQLConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	Charset  string `mapstructure:"charset"`
	// 连接池配置
	MaxOpenConns       int    `mapstructure:"max_open_conns"`
	MaxIdleConns       int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetimeStr string `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTimeStr string `mapstructure:"conn_max_idle_time"`

	// 解析后的时间，由 Load 函数填充
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// PostgreSQLConfig PostgreSQL 配置
type PostgreSQLConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"sslmode"`
	// 连接池配置
	MaxOpenConns       int    `mapstructure:"max_open_conns"`
	MaxIdleConns       int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetimeStr string `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTimeStr string `mapstructure:"conn_max_idle_time"`

	// 解析后的时间，由 Load 函数填充
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// MongoDBConfig MongoDB 配置
type MongoDBConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	URI         string `mapstructure:"uri"`
	Database    string `mapstructure:"database"`
	AuthSource  string `mapstructure:"auth_source"`
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	ReplicaSet  string `mapstructure:"replica_set"`
	MaxPoolSize uint64 `mapstructure:"max_pool_size"`
	MinPoolSize uint64 `mapstructure:"min_pool_size"`
	MaxIdleTime string `mapstructure:"max_idle_time"`
}

// KeepaAPIConfig Keepa API 配置
type KeepaAPIConfig struct {
	AccessKey        string           `mapstructure:"access_key"`
	Timeout          string           `mapstructure:"timeout"`            // 例如: "30s"
	PrintCurlCommand bool             `mapstructure:"print_curl_command"` // 是否打印 curl 命令（用于调试）
	PrintResponseBody bool            `mapstructure:"print_response_body"` // 是否打印 API 响应体（用于调试）
	Token            TokenConfig      `mapstructure:"token"`              // Token 管理配置
}

// TokenConfig Token 管理配置
type TokenConfig struct {
	MinTokensThreshold int    `mapstructure:"min_tokens_threshold"` // 最小token阈值
	MaxWaitTime        string `mapstructure:"max_wait_time"`       // 最大等待时间，例如: "60m"
	EnableRateLimit    bool   `mapstructure:"enable_rate_limit"`    // 是否启用速率限制

	// 解析后的时间，由 Load 函数填充
	MaxWaitTimeDuration time.Duration
}

// ServerConfig HTTP 服务器配置
type ServerConfig struct {
	Enabled bool   `mapstructure:"enabled"` // 是否启用服务器
	Host    string `mapstructure:"host"`    // 监听地址
	Port    int    `mapstructure:"port"`    // 监听端口
	Mode    string `mapstructure:"mode"`    // gin 模式: debug, release, test
}

// Load 加载配置
func Load(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(configPath)

	// 设置环境变量
	viper.SetEnvPrefix("KEEPA")
	viper.AutomaticEnv()

	// 设置默认值
	setDefaults()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 解析时间字符串
	if err := config.parseDurations(); err != nil {
		return nil, fmt.Errorf("failed to parse durations: %w", err)
	}

	return &config, nil
}

// parseDurations 解析时间字符串
func (c *Config) parseDurations() error {
	// 解析 MySQL 配置中的时间
	if c.Database.MySQL.ConnMaxLifetimeStr != "" {
		duration, err := time.ParseDuration(c.Database.MySQL.ConnMaxLifetimeStr)
		if err != nil {
			return fmt.Errorf("invalid mysql conn_max_lifetime: %w", err)
		}
		c.Database.MySQL.ConnMaxLifetime = duration
	}

	if c.Database.MySQL.ConnMaxIdleTimeStr != "" {
		duration, err := time.ParseDuration(c.Database.MySQL.ConnMaxIdleTimeStr)
		if err != nil {
			return fmt.Errorf("invalid mysql conn_max_idle_time: %w", err)
		}
		c.Database.MySQL.ConnMaxIdleTime = duration
	}

	// 解析 PostgreSQL 配置中的时间
	if c.Database.PostgreSQL.ConnMaxLifetimeStr != "" {
		duration, err := time.ParseDuration(c.Database.PostgreSQL.ConnMaxLifetimeStr)
		if err != nil {
			return fmt.Errorf("invalid postgresql conn_max_lifetime: %w", err)
		}
		c.Database.PostgreSQL.ConnMaxLifetime = duration
	}

	if c.Database.PostgreSQL.ConnMaxIdleTimeStr != "" {
		duration, err := time.ParseDuration(c.Database.PostgreSQL.ConnMaxIdleTimeStr)
		if err != nil {
			return fmt.Errorf("invalid postgresql conn_max_idle_time: %w", err)
		}
		c.Database.PostgreSQL.ConnMaxIdleTime = duration
	}

	// 解析 Token 配置中的时间
	if c.KeepaAPI.Token.MaxWaitTime != "" {
		duration, err := time.ParseDuration(c.KeepaAPI.Token.MaxWaitTime)
		if err != nil {
			return fmt.Errorf("invalid keepa_api.token.max_wait_time: %w", err)
		}
		c.KeepaAPI.Token.MaxWaitTimeDuration = duration
	}

	return nil
}

// setDefaults 设置默认配置值
func setDefaults() {
	// App 默认值
	viper.SetDefault("app.name", "keepa")
	viper.SetDefault("app.version", "1.0.0")
	viper.SetDefault("app.env", "development")

	// Scheduler 默认值
	viper.SetDefault("scheduler.default_timeout", "30m")
	viper.SetDefault("scheduler.location", "Asia/Shanghai")

	// Logger 默认值
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.format", "console")
	viper.SetDefault("logger.output_path", "logs/app.log")
	viper.SetDefault("logger.max_size", 100)
	viper.SetDefault("logger.max_backups", 3)
	viper.SetDefault("logger.max_age", 7)
	viper.SetDefault("logger.compress", true)

	// Database 默认值
	// MySQL
	viper.SetDefault("database.mysql.enabled", false)
	viper.SetDefault("database.mysql.host", "localhost")
	viper.SetDefault("database.mysql.port", 3306)
	viper.SetDefault("database.mysql.charset", "utf8mb4")
	viper.SetDefault("database.mysql.max_open_conns", 25)
	viper.SetDefault("database.mysql.max_idle_conns", 5)
	viper.SetDefault("database.mysql.conn_max_lifetime", "5m")
	viper.SetDefault("database.mysql.conn_max_idle_time", "10m")

	// PostgreSQL
	viper.SetDefault("database.postgresql.enabled", false)
	viper.SetDefault("database.postgresql.host", "localhost")
	viper.SetDefault("database.postgresql.port", 5432)
	viper.SetDefault("database.postgresql.sslmode", "disable")
	viper.SetDefault("database.postgresql.max_open_conns", 25)
	viper.SetDefault("database.postgresql.max_idle_conns", 5)
	viper.SetDefault("database.postgresql.conn_max_lifetime", "5m")
	viper.SetDefault("database.postgresql.conn_max_idle_time", "10m")

	// MongoDB
	viper.SetDefault("database.mongodb.enabled", false)
	viper.SetDefault("database.mongodb.uri", "mongodb://localhost:27017")
	viper.SetDefault("database.mongodb.auth_source", "admin")
	viper.SetDefault("database.mongodb.max_pool_size", 100)
	viper.SetDefault("database.mongodb.min_pool_size", 10)
	viper.SetDefault("database.mongodb.max_idle_time", "30m")

	// Keepa API 默认值
	viper.SetDefault("keepa_api.timeout", "30s")
	viper.SetDefault("keepa_api.print_curl_command", false)  // 默认不打印 curl 命令
	viper.SetDefault("keepa_api.print_response_body", false) // 默认不打印响应体
	viper.SetDefault("keepa_api.token.min_tokens_threshold", 5)
	viper.SetDefault("keepa_api.token.max_wait_time", "60m")
	viper.SetDefault("keepa_api.token.enable_rate_limit", true)

	// Server 默认值
	viper.SetDefault("server.enabled", true)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")
}

// GetDefaultTimeout 获取默认超时时间
func (c *Config) GetDefaultTimeout() (time.Duration, error) {
	return time.ParseDuration(c.Scheduler.DefaultTimeout)
}

// GetLocation 获取时区
func (c *Config) GetLocation() (*time.Location, error) {
	return time.LoadLocation(c.Scheduler.Location)
}
