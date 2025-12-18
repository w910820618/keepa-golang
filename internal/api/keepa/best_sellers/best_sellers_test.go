package best_sellers

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
	// 1. 创建 mock API client
	// 2. 创建 service 实例
	// 3. 测试 Fetch 方法
	// 4. 验证 API 调用

	t.Skip("TODO: implement test with mocks")

	ctx := context.Background()

	// Mock 客户端
	var client *api.Client
	logger := zap.NewNop()

	service := NewService(client, logger)

	params := RequestParams{
		Domain:   1,
		Category: "123456", // 类别 ID
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
			name: "basic request with category ID",
			params: RequestParams{
				Domain:   1,        // US
				Category: "172282", // Electronics category
			},
			wantErr: false,
		},
		{
			name: "request with category name",
			params: RequestParams{
				Domain:   1,
				Category: "Beauty",
			},
			wantErr: false,
		},
		{
			name: "request with range",
			params: RequestParams{
				Domain:   1,
				Category: "172282",
				Range:    intPtr(30), // 30-day average
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

func TestRequestParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  RequestParams
		wantErr bool
	}{
		{
			name: "valid basic params",
			params: RequestParams{
				Domain:   1,
				Category: "123456",
			},
			wantErr: false,
		},
		{
			name: "valid with range",
			params: RequestParams{
				Domain:   1,
				Category: "123456",
				Range:    intPtr(30),
			},
			wantErr: false,
		},
		{
			name: "valid with month and year",
			params: RequestParams{
				Domain:   1,
				Category: "123456",
				Month:    intPtr(6),
				Year:     intPtr(2024),
			},
			wantErr: false,
		},
		{
			name: "valid with variations",
			params: RequestParams{
				Domain:     1,
				Category:   "123456",
				Variations: intPtr(1),
			},
			wantErr: false,
		},
		{
			name: "valid with sublist",
			params: RequestParams{
				Domain:   1,
				Category: "123456",
				Sublist:  intPtr(1),
			},
			wantErr: false,
		},
		{
			name: "invalid domain",
			params: RequestParams{
				Domain:   7, // 无效的域名（巴西不支持）
				Category: "123456",
			},
			wantErr: true,
		},
		{
			name: "invalid domain - out of range",
			params: RequestParams{
				Domain:   99,
				Category: "123456",
			},
			wantErr: true,
		},
		{
			name: "missing category",
			params: RequestParams{
				Domain: 1,
			},
			wantErr: true,
		},
		{
			name: "invalid range value",
			params: RequestParams{
				Domain:   1,
				Category: "123456",
				Range:    intPtr(60), // 无效的值
			},
			wantErr: true,
		},
		{
			name: "invalid month",
			params: RequestParams{
				Domain:   1,
				Category: "123456",
				Month:    intPtr(13), // 无效的月份
			},
			wantErr: true,
		},
		{
			name: "month without year",
			params: RequestParams{
				Domain:   1,
				Category: "123456",
				Month:    intPtr(6),
			},
			wantErr: true,
		},
		{
			name: "year without month",
			params: RequestParams{
				Domain:   1,
				Category: "123456",
				Year:     intPtr(2024),
			},
			wantErr: true,
		},
		{
			name: "range with month/year conflict",
			params: RequestParams{
				Domain:   1,
				Category: "123456",
				Range:    intPtr(30),
				Month:    intPtr(6),
				Year:     intPtr(2024),
			},
			wantErr: true,
		},
		{
			name: "range with sublist conflict",
			params: RequestParams{
				Domain:   1,
				Category: "123456",
				Range:    intPtr(30),
				Sublist:  intPtr(1),
			},
			wantErr: true,
		},
		{
			name: "month/year with sublist conflict",
			params: RequestParams{
				Domain:   1,
				Category: "123456",
				Month:    intPtr(6),
				Year:     intPtr(2024),
				Sublist:  intPtr(1),
			},
			wantErr: true,
		},
		{
			name: "invalid variations value",
			params: RequestParams{
				Domain:     1,
				Category:   "123456",
				Variations: intPtr(2), // 无效的值
			},
			wantErr: true,
		},
		{
			name: "invalid sublist value",
			params: RequestParams{
				Domain:   1,
				Category: "123456",
				Sublist:  intPtr(2), // 无效的值
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RequestParams.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequestParams_ToQueryParams(t *testing.T) {
	tests := []struct {
		name   string
		params RequestParams
		want   map[string]string
	}{
		{
			name: "basic params",
			params: RequestParams{
				Domain:   1,
				Category: "123456",
			},
			want: map[string]string{
				"domain":   "1",
				"category": "123456",
			},
		},
		{
			name: "with all optional params",
			params: RequestParams{
				Domain:     1,
				Category:   "Beauty",
				Range:      intPtr(90),
				Variations: intPtr(1),
				Sublist:    intPtr(1),
			},
			want: map[string]string{
				"domain":     "1",
				"category":   "Beauty",
				"range":      "90",
				"variations": "1",
				"sublist":    "1",
			},
		},
		{
			name: "with month and year",
			params: RequestParams{
				Domain:   1,
				Category: "123456",
				Month:    intPtr(6),
				Year:     intPtr(2024),
			},
			want: map[string]string{
				"domain":   "1",
				"category": "123456",
				"month":    "6",
				"year":     "2024",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.params.ToQueryParams()

			// 检查必需参数
			if got["domain"] != tt.want["domain"] {
				t.Errorf("ToQueryParams() domain = %v, want %v", got["domain"], tt.want["domain"])
			}
			if got["category"] != tt.want["category"] {
				t.Errorf("ToQueryParams() category = %v, want %v", got["category"], tt.want["category"])
			}

			// 检查可选参数
			for key, wantValue := range tt.want {
				if key == "domain" || key == "category" {
					continue // 已经检查过了
				}
				if gotValue, ok := got[key]; !ok || gotValue != wantValue {
					t.Errorf("ToQueryParams() [%s] = %v, want %v", key, gotValue, wantValue)
				}
			}
		})
	}
}

func TestNewService(t *testing.T) {
	logger := zap.NewNop()

	service := NewService(nil, logger)
	if service == nil {
		t.Error("NewService() returned nil")
	}
	if service.client != nil {
		t.Error("NewService() client should be nil")
	}
	if service.logger != logger {
		t.Error("NewService() logger mismatch")
	}
}

// intPtr 辅助函数，用于创建 int 指针
func intPtr(i int) *int {
	return &i
}
