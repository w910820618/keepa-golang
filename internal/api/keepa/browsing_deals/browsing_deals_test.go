package browsing_deals

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"keepa/internal/api"
	"keepa/internal/config"
	"keepa/internal/logger"

	"go.uber.org/zap"
)

func TestService_Fetch(t *testing.T) {
	// TODO: 实现单元测试
	t.Skip("TODO: implement test")

	ctx := context.Background()
	logger := zap.NewNop()

	service := NewService(nil, logger)

	params := RequestParams{
		DomainID: 1,
	}

	_, err := service.Fetch(ctx, params)
	if err != nil {
		t.Errorf("Fetch() error = %v", err)
	}
}

// TestService_Fetch_Integration 集成测试：使用真实的 API key 调用 Keepa API
// 注意：此测试需要配置文件中包含有效的 keepa_api.access_key
// 如果没有配置 API key，测试会被跳过
func TestService_Fetch_Integration(t *testing.T) {
	// 获取项目根目录（从测试文件位置向上查找）
	_, testFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(testFile), "../../../..")

	// 加载配置文件
	configPath := filepath.Join(projectRoot, "configs")
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 检查是否有 API key
	if cfg.KeepaAPI.AccessKey == "" {
		t.Skip("Skipping integration test: keepa_api.access_key not configured in config.yaml")
	}

	// 解析超时时间
	timeout, err := time.ParseDuration(cfg.KeepaAPI.Timeout)
	if err != nil {
		timeout = 30 * time.Second
	}

	// 创建 logger
	loggerConfig := logger.LoggerConfig{
		Level:      cfg.Logger.Level,
		Format:     cfg.Logger.Format,
		OutputPath: cfg.Logger.OutputPath,
		MaxSize:    cfg.Logger.MaxSize,
		MaxBackups: cfg.Logger.MaxBackups,
		MaxAge:     cfg.Logger.MaxAge,
		Compress:   cfg.Logger.Compress,
	}
	zapLogger, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// 创建真实的 API 客户端
	client := api.NewClient(api.Config{
		AccessKey: cfg.KeepaAPI.AccessKey,
		Timeout:   timeout,
		Logger:    zapLogger,
	})

	// 创建服务
	service := NewService(client, zapLogger)

	// 测试用例
	tests := []struct {
		name    string
		params  RequestParams
		wantErr bool
	}{
		{
			name: "basic request",
			params: RequestParams{
				DomainID:   1,        // US
				PriceTypes: []int{1}, // New price
				DateRange:  0,        // Day
			},
			wantErr: false,
		},
		{
			name: "request with page",
			params: RequestParams{
				DomainID:   1,
				PriceTypes: []int{1},
				DateRange:  1, // Week
				Page:       intPtr(0),
			},
			wantErr: false,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := service.Fetch(ctx, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(data) == 0 {
					t.Error("Fetch() returned empty data")
				} else {
					// 格式化并打印 JSON 结果
					var jsonObj interface{}
					if err := json.Unmarshal(data, &jsonObj); err != nil {
						// 如果不是有效的 JSON，直接打印原始字符串
						t.Logf("Response (raw): %s", string(data))
					} else {
						// 格式化 JSON 并打印
						prettyJSON, err := json.MarshalIndent(jsonObj, "", "  ")
						if err != nil {
							t.Logf("Response (unformatted): %s", string(data))
						} else {
							t.Logf("Response JSON:\n%s", string(prettyJSON))
						}
					}
				}
			}
		})
	}
}

// intPtr 辅助函数，用于创建 int 指针
func intPtr(i int) *int {
	return &i
}
