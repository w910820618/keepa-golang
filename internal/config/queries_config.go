package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// KeepaQueryAPIConfig API 查询配置结构
type KeepaQueryAPIConfig struct {
	Enabled    bool                   `mapstructure:"enabled"`
	Collection string                 `mapstructure:"collection"`
	Tasks      []KeepaQueryTaskConfig `mapstructure:"tasks"`
}

// KeepaQueryTaskConfig 任务配置结构
type KeepaQueryTaskConfig struct {
	Name    string                 `mapstructure:"name"`
	Params  map[string]interface{} `mapstructure:"params"`
	Enabled bool                   `mapstructure:"enabled"`
}

// KeepaQueriesConfig 所有 API 查询配置
type KeepaQueriesConfig struct {
	BestSellers      KeepaQueryAPIConfig `mapstructure:"best_sellers"`
	CategoryLookup   KeepaQueryAPIConfig `mapstructure:"category_lookup"`
	BrowsingDeals    KeepaQueryAPIConfig `mapstructure:"browsing_deals"`
	Products         KeepaQueryAPIConfig `mapstructure:"products"`
	CategorySearches KeepaQueryAPIConfig `mapstructure:"category_searches"`
	LightningDeals   KeepaQueryAPIConfig `mapstructure:"lightning_deals"`
	MostRatedSellers KeepaQueryAPIConfig `mapstructure:"most_rated_sellers"`
	ProductFinder    KeepaQueryAPIConfig `mapstructure:"product_finder"`
	ProductSearches  KeepaQueryAPIConfig `mapstructure:"product_searches"`
	SellerInfo       KeepaQueryAPIConfig `mapstructure:"seller_information"`
}

// LoadKeepaQueriesConfig 加载 Keepa API 查询配置
func LoadKeepaQueriesConfig(configPath string, logger *zap.Logger) (*KeepaQueriesConfig, error) {
	viper.SetConfigName("keepa_api_queries")
	viper.SetConfigType("yaml")

	// 添加配置路径
	if configPath != "" {
		viper.AddConfigPath(configPath)
	}
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if logger != nil {
				logger.Warn("keepa_api_queries.yaml not found, using default empty config")
			}
			// 返回空配置
			return &KeepaQueriesConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read queries config file: %w", err)
	}

	var queriesConfig KeepaQueriesConfig
	if err := viper.Unmarshal(&queriesConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal queries config: %w", err)
	}

	if logger != nil {
		logger.Info("keepa API queries config loaded successfully",
			zap.String("config_file", viper.ConfigFileUsed()),
		)
	}

	return &queriesConfig, nil
}

// LoadKeepaQueriesConfigFromFile 从指定文件加载配置
func LoadKeepaQueriesConfigFromFile(filePath string, logger *zap.Logger) (*KeepaQueriesConfig, error) {
	viper.SetConfigType("yaml")
	viper.SetConfigFile(filePath)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if logger != nil {
			logger.Warn("queries config file not found, using default empty config",
				zap.String("file_path", filePath),
			)
		}
		return &KeepaQueriesConfig{}, nil
	}

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read queries config file: %w", err)
	}

	var queriesConfig KeepaQueriesConfig
	if err := viper.Unmarshal(&queriesConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal queries config: %w", err)
	}

	if logger != nil {
		logger.Info("keepa API queries config loaded successfully",
			zap.String("config_file", filePath),
		)
	}

	return &queriesConfig, nil
}
