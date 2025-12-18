package seller_information

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

// intPtr 返回整数的指针
func intPtr(i int) *int {
	return &i
}

func TestRequestParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  RequestParams
		wantErr bool
	}{
		{
			name: "valid params with required fields only",
			params: RequestParams{
				Domain:    1,
				SellerIDs: []string{"A2L77EE7U53NWQ"},
			},
			wantErr: false,
		},
		{
			name: "valid params with multiple seller IDs",
			params: RequestParams{
				Domain:    1,
				SellerIDs: []string{"A2L77EE7U53NWQ", "A1EXAMPLE"},
			},
			wantErr: false,
		},
		{
			name: "invalid domain",
			params: RequestParams{
				Domain:    7, // 无效的域名
				SellerIDs: []string{"A2L77EE7U53NWQ"},
			},
			wantErr: true,
		},
		{
			name: "empty seller IDs",
			params: RequestParams{
				Domain:    1,
				SellerIDs: []string{},
			},
			wantErr: true,
		},
		{
			name: "too many seller IDs",
			params: RequestParams{
				Domain:    1,
				SellerIDs: make([]string, 101), // 超过100个
			},
			wantErr: true,
		},
		{
			name: "empty seller ID in list",
			params: RequestParams{
				Domain:    1,
				SellerIDs: []string{"A2L77EE7U53NWQ", ""},
			},
			wantErr: true,
		},
		{
			name: "valid params with storefront",
			params: RequestParams{
				Domain:     1,
				SellerIDs:  []string{"A2L77EE7U53NWQ"},
				Storefront: intPtr(1),
			},
			wantErr: false,
		},
		{
			name: "invalid storefront value",
			params: RequestParams{
				Domain:     1,
				SellerIDs:  []string{"A2L77EE7U53NWQ"},
				Storefront: intPtr(2), // 无效值
			},
			wantErr: true,
		},
		{
			name: "storefront with batch request (invalid)",
			params: RequestParams{
				Domain:     1,
				SellerIDs:  []string{"A2L77EE7U53NWQ", "A1EXAMPLE"},
				Storefront: intPtr(1), // 不能与批量请求一起使用
			},
			wantErr: true,
		},
		{
			name: "update without storefront (invalid)",
			params: RequestParams{
				Domain:    1,
				SellerIDs: []string{"A2L77EE7U53NWQ"},
				Update:    intPtr(48), // 必须与 storefront 一起使用
			},
			wantErr: true,
		},
		{
			name: "valid params with storefront and update",
			params: RequestParams{
				Domain:     1,
				SellerIDs:  []string{"A2L77EE7U53NWQ"},
				Storefront: intPtr(1),
				Update:     intPtr(48),
			},
			wantErr: false,
		},
		{
			name: "invalid update value (negative)",
			params: RequestParams{
				Domain:     1,
				SellerIDs:  []string{"A2L77EE7U53NWQ"},
				Storefront: intPtr(1),
				Update:     intPtr(-1), // 无效值
			},
			wantErr: true,
		},
		{
			name: "seller IDs with whitespace (should be trimmed)",
			params: RequestParams{
				Domain:    1,
				SellerIDs: []string{" A2L77EE7U53NWQ ", " A1EXAMPLE "},
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
			name: "required params only",
			params: RequestParams{
				Domain:    1,
				SellerIDs: []string{"A2L77EE7U53NWQ"},
			},
			want: map[string]string{
				"domain": "1",
				"seller": "A2L77EE7U53NWQ",
			},
		},
		{
			name: "multiple seller IDs",
			params: RequestParams{
				Domain:    1,
				SellerIDs: []string{"A2L77EE7U53NWQ", "A1EXAMPLE"},
			},
			want: map[string]string{
				"domain": "1",
				"seller": "A2L77EE7U53NWQ,A1EXAMPLE",
			},
		},
		{
			name: "with storefront",
			params: RequestParams{
				Domain:     1,
				SellerIDs:  []string{"A2L77EE7U53NWQ"},
				Storefront: intPtr(1),
			},
			want: map[string]string{
				"domain":     "1",
				"seller":     "A2L77EE7U53NWQ",
				"storefront": "1",
			},
		},
		{
			name: "with storefront and update",
			params: RequestParams{
				Domain:     1,
				SellerIDs:  []string{"A2L77EE7U53NWQ"},
				Storefront: intPtr(1),
				Update:     intPtr(48),
			},
			want: map[string]string{
				"domain":     "1",
				"seller":     "A2L77EE7U53NWQ",
				"storefront": "1",
				"update":     "48",
			},
		},
		{
			name: "storefront set to 0 (should not be included)",
			params: RequestParams{
				Domain:     1,
				SellerIDs:  []string{"A2L77EE7U53NWQ"},
				Storefront: intPtr(0),
			},
			want: map[string]string{
				"domain": "1",
				"seller": "A2L77EE7U53NWQ",
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
	// TODO: 实现单元测试（需要 mock client）
	t.Skip("TODO: implement test with mocked client")

	ctx := context.Background()
	logger := zap.NewNop()

	service := NewService(nil, logger)

	params := RequestParams{
		SellerIDs: []string{"A2L77EE7U53NWQ"},
		Domain:    1,
	}

	_, err := service.Fetch(ctx, params)
	if err != nil {
		t.Errorf("Fetch() error = %v", err)
	}
}
