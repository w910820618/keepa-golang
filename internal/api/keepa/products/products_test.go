package products

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
			name: "valid params with ASINs only",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456"},
			},
			wantErr: false,
		},
		{
			name: "valid params with Codes only",
			params: RequestParams{
				Domain: 1,
				Codes:  []string{"1234567890123"},
			},
			wantErr: false,
		},
		{
			name: "invalid domain",
			params: RequestParams{
				Domain: 7, // 无效的域名
				ASINs:  []string{"B000123456"},
			},
			wantErr: true,
		},
		{
			name: "both ASINs and Codes (invalid)",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456"},
				Codes:  []string{"1234567890123"},
			},
			wantErr: true,
		},
		{
			name: "neither ASINs nor Codes (invalid)",
			params: RequestParams{
				Domain: 1,
			},
			wantErr: true,
		},
		{
			name: "too many ASINs",
			params: RequestParams{
				Domain: 1,
				ASINs:  make([]string, 101), // 超过100个
			},
			wantErr: true,
		},
		{
			name: "too many Codes",
			params: RequestParams{
				Domain: 1,
				Codes:  make([]string, 101), // 超过100个
			},
			wantErr: true,
		},
		{
			name: "invalid ASIN length",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B00012345"}, // 只有9个字符
			},
			wantErr: true,
		},
		{
			name: "valid params with all optional fields",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456"},
				Stats: &StatsValue{
					Days: intPtr(180),
				},
				Update:               intPtr(48),
				History:              intPtr(0),
				Days:                 intPtr(90),
				CodeLimit:            intPtr(10),
				Offers:               intPtr(40),
				OnlyLiveOffers:       intPtr(1),
				Rental:               intPtr(1),
				Videos:               intPtr(1),
				Aplus:                intPtr(1),
				Rating:               intPtr(1),
				Buybox:               intPtr(1),
				Stock:                intPtr(1),
				HistoricalVariations: intPtr(1),
			},
			wantErr: false,
		},
		{
			name: "stats with date range",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456"},
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
				ASINs:  []string{"B000123456"},
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
				ASINs:  []string{"B000123456"},
				Stats: &StatsValue{
					Days: intPtr(-1),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid update value (less than -1)",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456"},
				Update: intPtr(-2),
			},
			wantErr: true,
		},
		{
			name: "valid update value -1",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456"},
				Update: intPtr(-1),
			},
			wantErr: false,
		},
		{
			name: "valid update value 0",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456"},
				Update: intPtr(0),
			},
			wantErr: false,
		},
		{
			name: "invalid days value (negative)",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456"},
				Days:   intPtr(-1),
			},
			wantErr: true,
		},
		{
			name: "invalid offers value (too low)",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456"},
				Offers: intPtr(19),
			},
			wantErr: true,
		},
		{
			name: "invalid offers value (too high)",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456"},
				Offers: intPtr(101),
			},
			wantErr: true,
		},
		{
			name: "rental without offers (invalid)",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456"},
				Rental: intPtr(1),
			},
			wantErr: true,
		},
		{
			name: "stock without offers (invalid)",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456"},
				Stock:  intPtr(1),
			},
			wantErr: true,
		},
		{
			name: "multiple ASINs",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456", "B000789012", "B000345678"},
			},
			wantErr: false,
		},
		{
			name: "multiple Codes",
			params: RequestParams{
				Domain: 1,
				Codes:  []string{"1234567890123", "9876543210987"},
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
			name: "required params with ASINs only",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456"},
			},
			want: map[string]string{
				"domain": "1",
				"asin":   "B000123456",
			},
		},
		{
			name: "required params with Codes only",
			params: RequestParams{
				Domain: 1,
				Codes:  []string{"1234567890123"},
			},
			want: map[string]string{
				"domain": "1",
				"code":   "1234567890123",
			},
		},
		{
			name: "multiple ASINs",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456", "B000789012"},
			},
			want: map[string]string{
				"domain": "1",
				"asin":   "B000123456,B000789012",
			},
		},
		{
			name: "with all optional params",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456"},
				Stats: &StatsValue{
					Days: intPtr(180),
				},
				Update:               intPtr(48),
				History:              intPtr(0),
				Days:                 intPtr(90),
				CodeLimit:            intPtr(10),
				Offers:               intPtr(40),
				OnlyLiveOffers:       intPtr(1),
				Rental:               intPtr(1),
				Videos:               intPtr(1),
				Aplus:                intPtr(1),
				Rating:               intPtr(1),
				Buybox:               intPtr(1),
				Stock:                intPtr(1),
				HistoricalVariations: intPtr(1),
			},
			want: map[string]string{
				"domain":                "1",
				"asin":                  "B000123456",
				"stats":                 "180",
				"update":                "48",
				"history":               "0",
				"days":                  "90",
				"code-limit":            "10",
				"offers":                "40",
				"only-live-offers":      "1",
				"rental":                "1",
				"videos":                "1",
				"aplus":                 "1",
				"rating":                "1",
				"buybox":                "1",
				"stock":                 "1",
				"historical-variations": "1",
			},
		},
		{
			name: "with stats date range",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456"},
				Stats: &StatsValue{
					DateRange: stringPtr("2015-10-20,2015-12-24"),
				},
			},
			want: map[string]string{
				"domain": "1",
				"asin":   "B000123456",
				"stats":  "2015-10-20,2015-12-24",
			},
		},
		{
			name: "with history=1 (should not appear in params, as default is 1)",
			params: RequestParams{
				Domain:  1,
				ASINs:   []string{"B000123456"},
				History: intPtr(1),
			},
			want: map[string]string{
				"domain": "1",
				"asin":   "B000123456",
			},
		},
		{
			name: "with update=-1",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456"},
				Update: intPtr(-1),
			},
			want: map[string]string{
				"domain": "1",
				"asin":   "B000123456",
				"update": "-1",
			},
		},
		{
			name: "with update=0",
			params: RequestParams{
				Domain: 1,
				ASINs:  []string{"B000123456"},
				Update: intPtr(0),
			},
			want: map[string]string{
				"domain": "1",
				"asin":   "B000123456",
				"update": "0",
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
		ASINs:  []string{"B000123456", "B000789012"},
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
