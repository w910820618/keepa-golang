package most_rated_sellers

import (
	"context"
	"testing"

	"keepa/internal/api"

	"go.uber.org/zap"
)

func TestService_FetchRaw(t *testing.T) {
	// TODO: 实现单元测试
	// 1. 创建 mock API client
	// 2. 创建 service 实例
	// 3. 测试 FetchRaw 方法
	// 4. 验证 API 调用

	t.Skip("TODO: implement test with mocks")

	ctx := context.Background()

	// Mock 客户端
	var client *api.Client
	logger := zap.NewNop()

	service := NewService(client, logger)

	params := RequestParams{
		Domain: 1, // com
	}

	_, err := service.FetchRaw(ctx, params)
	if err != nil {
		t.Errorf("FetchRaw() error = %v", err)
	}
}

func TestRequestParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  RequestParams
		wantErr bool
	}{
		{
			name: "valid domain - com",
			params: RequestParams{
				Domain: 1,
			},
			wantErr: false,
		},
		{
			name: "valid domain - co.uk",
			params: RequestParams{
				Domain: 2,
			},
			wantErr: false,
		},
		{
			name: "valid domain - de",
			params: RequestParams{
				Domain: 3,
			},
			wantErr: false,
		},
		{
			name: "valid domain - fr",
			params: RequestParams{
				Domain: 4,
			},
			wantErr: false,
		},
		{
			name: "valid domain - co.jp",
			params: RequestParams{
				Domain: 5,
			},
			wantErr: false,
		},
		{
			name: "valid domain - ca",
			params: RequestParams{
				Domain: 6,
			},
			wantErr: false,
		},
		{
			name: "valid domain - it",
			params: RequestParams{
				Domain: 8,
			},
			wantErr: false,
		},
		{
			name: "valid domain - es",
			params: RequestParams{
				Domain: 9,
			},
			wantErr: false,
		},
		{
			name: "valid domain - in",
			params: RequestParams{
				Domain: 10,
			},
			wantErr: false,
		},
		{
			name: "valid domain - com.mx",
			params: RequestParams{
				Domain: 11,
			},
			wantErr: false,
		},
		{
			name: "invalid domain - Brazil (not supported)",
			params: RequestParams{
				Domain: 7, // 巴西不支持
			},
			wantErr: true,
		},
		{
			name: "invalid domain - out of range",
			params: RequestParams{
				Domain: 99,
			},
			wantErr: true,
		},
		{
			name: "invalid domain - zero",
			params: RequestParams{
				Domain: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid domain - negative",
			params: RequestParams{
				Domain: -1,
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
			name: "domain 1 (com)",
			params: RequestParams{
				Domain: 1,
			},
			want: map[string]string{
				"domain": "1",
			},
		},
		{
			name: "domain 2 (co.uk)",
			params: RequestParams{
				Domain: 2,
			},
			want: map[string]string{
				"domain": "2",
			},
		},
		{
			name: "domain 11 (com.mx)",
			params: RequestParams{
				Domain: 11,
			},
			want: map[string]string{
				"domain": "11",
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

			// 确保没有其他参数
			if len(got) != len(tt.want) {
				t.Errorf("ToQueryParams() returned %d params, want %d", len(got), len(tt.want))
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
