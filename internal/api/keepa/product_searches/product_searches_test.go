package product_searches

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestRequestParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  RequestParams
		wantErr bool
	}{
		{
			name: "valid params with required fields only",
			params: RequestParams{
				Domain: 1,
				Term:   "laptop",
			},
			wantErr: false,
		},
		{
			name: "invalid domain",
			params: RequestParams{
				Domain: 7, // 无效的域名
				Term:   "laptop",
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
			name: "valid params with all optional fields",
			params: RequestParams{
				Domain:    1,
				Term:      "laptop",
				AsinsOnly: intPtr(1),
				Page:      intPtr(0),
				Stats: &StatsValue{
					Days: intPtr(180),
				},
				Update:  intPtr(48),
				History: intPtr(0),
				Rating:  intPtr(1),
			},
			wantErr: false,
		},
		{
			name: "invalid page value",
			params: RequestParams{
				Domain: 1,
				Term:   "laptop",
				Page:   intPtr(10), // 超出范围
			},
			wantErr: true,
		},
		{
			name: "invalid asins-only value",
			params: RequestParams{
				Domain:    1,
				Term:      "laptop",
				AsinsOnly: intPtr(2), // 无效值
			},
			wantErr: true,
		},
		{
			name: "stats with date range",
			params: RequestParams{
				Domain: 1,
				Term:   "laptop",
				Stats: &StatsValue{
					DateRange: stringPtr("2015-10-20,2015-12-24"),
				},
			},
			wantErr: false,
		},
		{
			name: "stats with both days and date range (invalid)",
			params: RequestParams{
				Domain: 1,
				Term:   "laptop",
				Stats: &StatsValue{
					Days:      intPtr(180),
					DateRange: stringPtr("2015-10-20,2015-12-24"),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid stats days (negative)",
			params: RequestParams{
				Domain: 1,
				Term:   "laptop",
				Stats: &StatsValue{
					Days: intPtr(-1),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid update value (negative)",
			params: RequestParams{
				Domain: 1,
				Term:   "laptop",
				Update: intPtr(-1),
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
			name: "required params only",
			params: RequestParams{
				Domain: 1,
				Term:   "laptop",
			},
			want: map[string]string{
				"domain": "1",
				"type":   "product",
				"term":   "laptop",
			},
		},
		{
			name: "with all optional params",
			params: RequestParams{
				Domain:    1,
				Term:      "laptop",
				AsinsOnly: intPtr(1),
				Page:      intPtr(2),
				Stats: &StatsValue{
					Days: intPtr(180),
				},
				Update:  intPtr(48),
				History: intPtr(0),
				Rating:  intPtr(1),
			},
			want: map[string]string{
				"domain":     "1",
				"type":       "product",
				"term":       "laptop",
				"asins-only": "1",
				"page":       "2",
				"stats":      "180",
				"update":     "48",
				"history":    "0",
				"rating":     "1",
			},
		},
		{
			name: "with stats date range",
			params: RequestParams{
				Domain: 1,
				Term:   "laptop",
				Stats: &StatsValue{
					DateRange: stringPtr("2015-10-20,2015-12-24"),
				},
			},
			want: map[string]string{
				"domain": "1",
				"type":   "product",
				"term":   "laptop",
				"stats":  "2015-10-20,2015-12-24",
			},
		},
		{
			name: "with history=1 (should not appear in params, as default is 1)",
			params: RequestParams{
				Domain:  1,
				Term:    "laptop",
				History: intPtr(1),
			},
			want: map[string]string{
				"domain": "1",
				"type":   "product",
				"term":   "laptop",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.params.ToQueryParams()
			if len(got) != len(tt.want) {
				t.Errorf("ToQueryParams() length = %d, want %d", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("ToQueryParams()[%s] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}

func TestService_Fetch(t *testing.T) {
	// TODO: 实现集成测试（需要真实的 client）
	t.Skip("TODO: implement integration test")

	ctx := context.Background()
	logger := zap.NewNop()

	service := NewService(nil, logger)

	params := RequestParams{
		Domain: 1,
		Term:   "laptop",
	}

	_, err := service.Fetch(ctx, params)
	if err != nil {
		t.Errorf("Fetch() error = %v", err)
	}
}

// 辅助函数
func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}
