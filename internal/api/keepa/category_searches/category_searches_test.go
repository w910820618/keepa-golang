package category_searches

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
	logger := zap.NewNop()

	// Mock 客户端
	var client *api.Client

	service := NewService(client, logger)

	params := RequestParams{
		Domain: 1,
		Term:   "electronics",
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
			name: "basic request with search term",
			params: RequestParams{
				Domain: 1, // US
				Term:   "electronics",
			},
			wantErr: false,
		},
		{
			name: "request with multiple keywords",
			params: RequestParams{
				Domain: 1,
				Term:   "laptop computer",
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
				Domain: 1,
				Term:   "electronics",
			},
			wantErr: false,
		},
		{
			name: "valid with multiple keywords",
			params: RequestParams{
				Domain: 1,
				Term:   "electronics computers",
			},
			wantErr: false,
		},
		{
			name: "valid with three character keyword",
			params: RequestParams{
				Domain: 1,
				Term:   "abc",
			},
			wantErr: false,
		},
		{
			name: "invalid domain",
			params: RequestParams{
				Domain: 7, // 无效的域名
				Term:   "electronics",
			},
			wantErr: true,
		},
		{
			name: "empty term",
			params: RequestParams{
				Domain: 1,
				Term:   "",
			},
			wantErr: true,
		},
		{
			name: "term with only whitespace",
			params: RequestParams{
				Domain: 1,
				Term:   "   ",
			},
			wantErr: true,
		},
		{
			name: "keyword too short (less than 3 characters)",
			params: RequestParams{
				Domain: 1,
				Term:   "ab",
			},
			wantErr: true,
		},
		{
			name: "one keyword too short in multiple keywords",
			params: RequestParams{
				Domain: 1,
				Term:   "electronics ab",
			},
			wantErr: true,
		},
		{
			name: "valid domain 11",
			params: RequestParams{
				Domain: 11,
				Term:   "electronics",
			},
			wantErr: false,
		},
		{
			name: "valid with long term",
			params: RequestParams{
				Domain: 1,
				Term:   "electronics computers laptops accessories",
			},
			wantErr: false,
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
		name           string
		params         RequestParams
		expectedDomain string
		expectedType   string
		expectedTerm   string
	}{
		{
			name: "basic params",
			params: RequestParams{
				Domain: 1,
				Term:   "electronics",
			},
			expectedDomain: "1",
			expectedType:   "category",
			expectedTerm:   "electronics",
		},
		{
			name: "with multiple keywords",
			params: RequestParams{
				Domain: 2,
				Term:   "electronics computers",
			},
			expectedDomain: "2",
			expectedType:   "category",
			expectedTerm:   "electronics computers",
		},
		{
			name: "with special characters",
			params: RequestParams{
				Domain: 1,
				Term:   "electronics & computers",
			},
			expectedDomain: "1",
			expectedType:   "category",
			expectedTerm:   "electronics & computers",
		},
		{
			name: "domain 11",
			params: RequestParams{
				Domain: 11,
				Term:   "electronics",
			},
			expectedDomain: "11",
			expectedType:   "category",
			expectedTerm:   "electronics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryParams := tt.params.ToQueryParams()

			if queryParams["domain"] != tt.expectedDomain {
				t.Errorf("ToQueryParams() domain = %v, want %v", queryParams["domain"], tt.expectedDomain)
			}

			if queryParams["type"] != tt.expectedType {
				t.Errorf("ToQueryParams() type = %v, want %v", queryParams["type"], tt.expectedType)
			}

			if queryParams["term"] != tt.expectedTerm {
				t.Errorf("ToQueryParams() term = %v, want %v", queryParams["term"], tt.expectedTerm)
			}
		})
	}
}
