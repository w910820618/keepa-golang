package category_lookup

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

func intPtr(i int) *int {
	return &i
}

func TestService_Fetch(t *testing.T) {
	// TODO: 实现单元测试
	// 1. 创建 mock API client
	// 2. 创建 service 实例
	// 3. 测试 Fetch 方法
	// 4. 验证 API 调用

	t.Skip("TODO: implement test with mocks")

	ctx := context.Background()
	logger := zap.NewNop()

	// Mock 客户端
	var client *api.Client

	service := NewService(client, logger)

	params := RequestParams{
		Domain:   1,
		Category: []int{123456},
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
				Domain:   1,             // US
				Category: []int{172282}, // Electronics category
			},
			wantErr: false,
		},
		{
			name: "request with root category",
			params: RequestParams{
				Domain:   1,
				Category: []int{0}, // Root category
			},
			wantErr: false,
		},
		{
			name: "request with includeParents",
			params: RequestParams{
				Domain:         1,
				Category:       []int{172282},
				IncludeParents: intPtr(1),
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
			name: "valid basic params with single category",
			params: RequestParams{
				Domain:   1,
				Category: []int{123456},
			},
			wantErr: false,
		},
		{
			name: "valid with root category",
			params: RequestParams{
				Domain:   1,
				Category: []int{0},
			},
			wantErr: false,
		},
		{
			name: "valid with multiple categories",
			params: RequestParams{
				Domain:   1,
				Category: []int{123456, 789012, 345678},
			},
			wantErr: false,
		},
		{
			name: "valid with includeParents",
			params: RequestParams{
				Domain:         1,
				Category:       []int{123456},
				IncludeParents: intPtr(1),
			},
			wantErr: false,
		},
		{
			name: "invalid domain",
			params: RequestParams{
				Domain:   7, // 无效的域名
				Category: []int{123456},
			},
			wantErr: true,
		},
		{
			name: "empty category",
			params: RequestParams{
				Domain:   1,
				Category: []int{},
			},
			wantErr: true,
		},
		{
			name: "too many categories",
			params: RequestParams{
				Domain:   1,
				Category: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}, // 超过10个
			},
			wantErr: true,
		},
		{
			name: "invalid category ID (negative)",
			params: RequestParams{
				Domain:   1,
				Category: []int{-1},
			},
			wantErr: true,
		},
		{
			name: "invalid: zero mixed with other IDs",
			params: RequestParams{
				Domain:   1,
				Category: []int{0, 123456}, // 0 不能与其他 ID 混合
			},
			wantErr: true,
		},
		{
			name: "invalid includeParents",
			params: RequestParams{
				Domain:         1,
				Category:       []int{123456},
				IncludeParents: intPtr(2), // 必须是0或1
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
		name            string
		params          RequestParams
		expectedDomain  string
		expectedCat     string
		expectedParents string
	}{
		{
			name: "basic params",
			params: RequestParams{
				Domain:   1,
				Category: []int{123456},
			},
			expectedDomain:  "1",
			expectedCat:     "123456",
			expectedParents: "",
		},
		{
			name: "with multiple categories",
			params: RequestParams{
				Domain:   2,
				Category: []int{123, 456, 789},
			},
			expectedDomain:  "2",
			expectedCat:     "123,456,789",
			expectedParents: "",
		},
		{
			name: "with includeParents",
			params: RequestParams{
				Domain:         1,
				Category:       []int{123456},
				IncludeParents: intPtr(1),
			},
			expectedDomain:  "1",
			expectedCat:     "123456",
			expectedParents: "1",
		},
		{
			name: "with root category and includeParents",
			params: RequestParams{
				Domain:         1,
				Category:       []int{0},
				IncludeParents: intPtr(0),
			},
			expectedDomain:  "1",
			expectedCat:     "0",
			expectedParents: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryParams := tt.params.ToQueryParams()

			if queryParams["domain"] != tt.expectedDomain {
				t.Errorf("ToQueryParams() domain = %v, want %v", queryParams["domain"], tt.expectedDomain)
			}

			if queryParams["category"] != tt.expectedCat {
				t.Errorf("ToQueryParams() category = %v, want %v", queryParams["category"], tt.expectedCat)
			}

			parents := queryParams["parents"]
			if parents != tt.expectedParents {
				t.Errorf("ToQueryParams() parents = %v, want %v", parents, tt.expectedParents)
			}
		})
	}
}
