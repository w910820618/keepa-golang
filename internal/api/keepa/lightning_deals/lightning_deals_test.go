package lightning_deals

import (
	"context"
	"testing"

	"keepa/internal/api"

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
	}

	_, err := service.Fetch(ctx, params)
	if err != nil {
		t.Errorf("Fetch() error = %v", err)
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
			},
			wantErr: false,
		},
		{
			name: "valid with ASIN",
			params: RequestParams{
				Domain: 1,
				ASIN:   "B012345678",
			},
			wantErr: false,
		},
		{
			name: "valid with state",
			params: RequestParams{
				Domain: 1,
				State:  StateAvailable,
			},
			wantErr: false,
		},
		{
			name: "valid with ASIN and state",
			params: RequestParams{
				Domain: 1,
				ASIN:   "B012345678",
				State:  StateAvailable,
			},
			wantErr: false,
		},
		{
			name: "invalid domain",
			params: RequestParams{
				Domain: 7, // 无效的域名
			},
			wantErr: true,
		},
		{
			name: "invalid domain - 0",
			params: RequestParams{
				Domain: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid domain - 12",
			params: RequestParams{
				Domain: 12,
			},
			wantErr: true,
		},
		{
			name: "invalid ASIN - too short",
			params: RequestParams{
				Domain: 1,
				ASIN:   "B01234567",
			},
			wantErr: true,
		},
		{
			name: "invalid ASIN - too long",
			params: RequestParams{
				Domain: 1,
				ASIN:   "B0123456789",
			},
			wantErr: true,
		},
		{
			name: "invalid ASIN - with spaces",
			params: RequestParams{
				Domain: 1,
				ASIN:   "B01234567 ",
			},
			wantErr: true,
		},
		{
			name: "valid ASIN - with spaces trimmed",
			params: RequestParams{
				Domain: 1,
				ASIN:   " B012345678 ",
			},
			wantErr: false,
		},
		{
			name: "invalid state",
			params: RequestParams{
				Domain: 1,
				State:  "INVALID_STATE",
			},
			wantErr: true,
		},
		{
			name: "valid all states",
			params: RequestParams{
				Domain: 1,
				State:  StateAvailable,
			},
			wantErr: false,
		},
		{
			name: "valid state WAITLIST",
			params: RequestParams{
				Domain: 1,
				State:  StateWaitlist,
			},
			wantErr: false,
		},
		{
			name: "valid state SOLDOUT",
			params: RequestParams{
				Domain: 1,
				State:  StateSoldout,
			},
			wantErr: false,
		},
		{
			name: "valid state WAITLISTFULL",
			params: RequestParams{
				Domain: 1,
				State:  StateWaitlistFull,
			},
			wantErr: false,
		},
		{
			name: "valid state EXPIRED",
			params: RequestParams{
				Domain: 1,
				State:  StateExpired,
			},
			wantErr: false,
		},
		{
			name: "valid state SUPPRESSED",
			params: RequestParams{
				Domain: 1,
				State:  StateSuppressed,
			},
			wantErr: false,
		},
		{
			name: "valid all domains",
			params: RequestParams{
				Domain: 2, // co.uk
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
		name   string
		params RequestParams
		want   map[string]string
	}{
		{
			name: "basic params",
			params: RequestParams{
				Domain: 1,
			},
			want: map[string]string{
				"domain": "1",
			},
		},
		{
			name: "with ASIN",
			params: RequestParams{
				Domain: 1,
				ASIN:   "B012345678",
			},
			want: map[string]string{
				"domain": "1",
				"asin":   "B012345678",
			},
		},
		{
			name: "with state",
			params: RequestParams{
				Domain: 1,
				State:  StateAvailable,
			},
			want: map[string]string{
				"domain": "1",
				"state":  "AVAILABLE",
			},
		},
		{
			name: "with ASIN and state",
			params: RequestParams{
				Domain: 1,
				ASIN:   "B012345678",
				State:  StateSoldout,
			},
			want: map[string]string{
				"domain": "1",
				"asin":   "B012345678",
				"state":  "SOLDOUT",
			},
		},
		{
			name: "different domain",
			params: RequestParams{
				Domain: 2,
			},
			want: map[string]string{
				"domain": "2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.params.ToQueryParams()
			if len(got) != len(tt.want) {
				t.Errorf("ToQueryParams() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("ToQueryParams() [%s] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}
